package message_test

import (
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/Trendyol/go-pq-cdc/pq/message"
	"github.com/Trendyol/go-pq-cdc/pq/message/format"
	"github.com/Trendyol/go-pq-cdc/pq/message/tuple"
)

// Hidden tests: every strict prefix of a well-formed pgoutput message must be
// rejected with an error, and the well-formed message itself must still decode.

const relOID = 16420

type msgBuf []byte

func (b msgBuf) u8(v byte) msgBuf     { return append(b, v) }
func (b msgBuf) u16(v uint16) msgBuf  { return binary.BigEndian.AppendUint16(b, v) }
func (b msgBuf) u32(v uint32) msgBuf  { return binary.BigEndian.AppendUint32(b, v) }
func (b msgBuf) u64(v uint64) msgBuf  { return binary.BigEndian.AppendUint64(b, v) }
func (b msgBuf) cstr(s string) msgBuf { return append(append(b, s...), 0) }
func (b msgBuf) text(s string) msgBuf { return b.u8('t').u32(uint32(len(s))).raw(s) }
func (b msgBuf) raw(s string) msgBuf  { return append(b, s...) }

func xidPrefix(kind byte, streamed bool) msgBuf {
	b := msgBuf{kind}
	if streamed {
		b = b.u32(901)
	}
	return b
}

func relationMsg(streamed bool) []byte {
	return xidPrefix('R', streamed).u32(relOID).cstr("public").cstr("shift_orders").u8('d').u16(3).
		u8(1).cstr("id").u32(23).u32(0xFFFFFFFF).
		u8(0).cstr("courier").u32(25).u32(0xFFFFFFFF).
		u8(0).cstr("note").u32(25).u32(0xFFFFFFFF)
}

func newTuple(b msgBuf, kind byte) msgBuf {
	return b.u8(kind).u16(3).text("42").text("c-17").u8('n')
}

func samples(streamed bool) map[string][]byte {
	s := map[string][]byte{
		"relation":     relationMsg(streamed),
		"insert":       newTuple(xidPrefix('I', streamed).u32(relOID), 'N'),
		"update/new":   newTuple(xidPrefix('U', streamed).u32(relOID), 'N'),
		"update/key":   newTuple(xidPrefix('U', streamed).u32(relOID).u8('K').u16(3).text("42").u8('n').u8('n'), 'N'),
		"update/old":   xidPrefix('U', streamed).u32(relOID).u8('O').u16(3).text("42").text("c-9").text("x").u8('N').u16(3).text("42").text("c-17").u8('u'),
		"delete/key":   xidPrefix('D', streamed).u32(relOID).u8('K').u16(3).text("42").u8('n').u8('n'),
		"delete/old":   xidPrefix('D', streamed).u32(relOID).u8('O').u16(3).text("42").text("c-17").u8('n'),
		"truncate":     xidPrefix('T', streamed).u32(2).u8(1).u32(relOID).u32(relOID),
	}
	if !streamed {
		s["begin"] = msgBuf{'B'}.u64(0x16B3748).u64(0x2A0000).u32(731)
		s["commit"] = msgBuf{'C'}.u8(0).u64(0x16B3748).u64(0x16B3790).u64(0x2A0000)
		s["stream-start"] = msgBuf{'S'}.u32(901).u8(1)
		s["stream-abort"] = msgBuf{'A'}.u32(901).u32(902)
		s["stream-commit"] = msgBuf{'c'}.u32(901).u8(0).u64(1).u64(2).u64(3)
	}
	return s
}

func decode(t *testing.T, data []byte, streamed bool, rels map[uint32]*format.Relation) (msg any, err error) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("decoder panicked on %d-byte input %x: %v", len(data), data, r)
		}
	}()
	return message.New(data, streamed, time.Unix(0, 0), rels)
}

func knownRelations(t *testing.T, streamed bool) map[uint32]*format.Relation {
	t.Helper()
	rels := map[uint32]*format.Relation{}
	if _, err := decode(t, relationMsg(streamed), streamed, rels); err != nil {
		t.Fatalf("well-formed relation message rejected: %v", err)
	}
	return rels
}

func TestHiddenWellFormedMessagesStillDecode(t *testing.T) {
	for _, streamed := range []bool{false, true} {
		for name, data := range samples(streamed) {
			t.Run(fmt.Sprintf("%s/streamed=%v", name, streamed), func(t *testing.T) {
				rels := knownRelations(t, streamed)
				msg, err := decode(t, data, streamed, rels)
				if err != nil {
					t.Fatalf("well-formed message rejected: %v", err)
				}
				if msg == nil {
					t.Fatal("well-formed message decoded to nil")
				}
			})
		}
	}
}

func TestHiddenDecodedValuesUnchanged(t *testing.T) {
	rels := knownRelations(t, false)
	rel := rels[relOID]
	if rel.Namespace != "public" || rel.Name != "shift_orders" || len(rel.Columns) != 3 || rel.Columns[2].Name != "note" {
		t.Fatalf("relation decoded wrongly: %+v", rel)
	}

	msg, err := decode(t, samples(false)["insert"], false, rels)
	if err != nil {
		t.Fatal(err)
	}
	ins := msg.(*format.Insert)
	if ins.Decoded["id"] != int32(42) || ins.Decoded["courier"] != "c-17" || ins.Decoded["note"] != nil {
		t.Fatalf("insert decoded wrongly: %#v", ins.Decoded)
	}

	msg, err = decode(t, samples(false)["update/old"], false, rels)
	if err != nil {
		t.Fatal(err)
	}
	upd := msg.(*format.Update)
	// the unchanged TOASTed column is taken from the old tuple
	if upd.NewDecoded["courier"] != "c-17" || upd.NewDecoded["note"] != "x" || upd.OldDecoded["courier"] != "c-9" {
		t.Fatalf("update decoded wrongly: new=%#v old=%#v", upd.NewDecoded, upd.OldDecoded)
	}

	msg, err = decode(t, samples(false)["begin"], false, rels)
	if err != nil {
		t.Fatal(err)
	}
	if b := msg.(*format.Begin); b.Xid != 731 || uint64(b.FinalLSN) != 0x16B3748 {
		t.Fatalf("begin decoded wrongly: %+v", b)
	}

	msg, err = decode(t, samples(false)["commit"], false, rels)
	if err != nil {
		t.Fatal(err)
	}
	if c := msg.(*format.Commit); uint64(c.CommitLSN) != 0x16B3748 || uint64(c.TransactionEndLSN) != 0x16B3790 {
		t.Fatalf("commit decoded wrongly: %+v", c)
	}
}

func TestHiddenEveryTruncatedPrefixIsAnError(t *testing.T) {
	for _, streamed := range []bool{false, true} {
		for name, data := range samples(streamed) {
			t.Run(fmt.Sprintf("%s/streamed=%v", name, streamed), func(t *testing.T) {
				rels := knownRelations(t, streamed)
				for n := 0; n < len(data); n++ {
					prefix := append([]byte(nil), data[:n]...)
					if _, err := decode(t, prefix, streamed, rels); err == nil {
						t.Fatalf("%d of %d bytes: truncated message accepted", n, len(data))
					}
				}
			})
		}
	}
}

func TestHiddenTruncatedRelationLeavesCacheUntouched(t *testing.T) {
	data := relationMsg(false)
	rels := map[uint32]*format.Relation{}
	for n := 1; n < len(data); n++ {
		_, _ = decode(t, data[:n], false, rels)
	}
	if len(rels) != 0 {
		t.Fatalf("truncated relation messages must not be cached, got %d entries", len(rels))
	}
}

func TestHiddenUpdateOldTupleShorterThanNewTuple(t *testing.T) {
	rels := knownRelations(t, false)
	// old tuple carries one column, new tuple has an unchanged TOASTed third column
	data := msgBuf{'U'}.u32(relOID).u8('O').u16(1).text("42").u8('N').u16(3).text("42").text("c-17").u8('u')
	_, _ = decode(t, data, false, rels)
}

func TestHiddenDirectDecodersRejectTruncatedInput(t *testing.T) {
	cases := map[string]func() error{
		"begin/20":          func() error { _, err := format.NewBegin(make([]byte, 20)); return err },
		"commit/25":         func() error { _, err := format.NewCommit(make([]byte, 25)); return err },
		"relation/no-repl":  func() error { _, err := format.NewRelation(msgBuf{'R'}.u32(1).cstr("s").cstr("t"), false); return err },
		"relation/no-typmod": func() error {
			_, err := format.NewRelation(msgBuf{'R'}.u32(1).cstr("s").cstr("t").u8('d').u16(1).u8(0).cstr("c").u32(25).u16(0), false)
			return err
		},
		"tuple/no-count":     func() error { _, err := tuple.NewData([]byte{'N', 0}, 'N', 0); return err },
		"tuple/no-length":    func() error { _, err := tuple.NewData([]byte{'N', 0, 1, 'b', 0, 0}, 'N', 0); return err },
		"tuple/short-binary": func() error { _, err := tuple.NewData([]byte{'N', 0, 1, 'b', 0, 0, 0, 3, 1, 2}, 'N', 0); return err },
		"tuple/missing-col":  func() error { _, err := tuple.NewData([]byte{'X', 'N', 0, 3, 'n', 'u'}, 'N', 1); return err },
		"tuple/offset-past":  func() error { _, err := tuple.NewData([]byte{'N'}, 'N', 4); return err },
		"insert/no-tuple":    func() error { _, err := format.NewInsert(msgBuf{'I'}.u32(relOID), false, nil, time.Time{}); return err },
		"delete/no-tuple":    func() error { _, err := format.NewDelete(msgBuf{'D'}.u32(relOID).u8('K'), false, nil, time.Time{}); return err },
	}
	for name, fn := range cases {
		t.Run(name, func(t *testing.T) {
			var err error
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("panicked: %v", r)
					}
				}()
				err = fn()
			}()
			if err == nil {
				t.Fatal("truncated input accepted")
			}
		})
	}
}
