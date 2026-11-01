// Hidden tests: never shown to the agent.
import * as YAML from 'yaml'
import { _map, _seq } from '../_utils.ts'

const js = (src: string) => {
  const doc = YAML.parseDocument(src)
  expect(doc.errors).toHaveLength(0)
  return doc.toJS({ mapAsMap: true })
}

describe('hidden: flow collection keys after blank lines and comments', () => {
  test.each([
    ['comment line', '1: 2\n# c\n[3]: 4'],
    ['two blank lines', '1: 2\n\n\n[3]: 4'],
    ['comment then blank line', '1: 2\n# c\n\n[3]: 4'],
    ['blank line then comment', '1: 2\n\n# c\n[3]: 4']
  ])('sequence key after a %s', (_, src) => {
    const doc = YAML.parseDocument(src)
    expect(doc.errors).toHaveLength(0)
    expect(doc.value).toMatchObject(
      _map([
        [1, 2],
        [_seq(3), 4]
      ])
    )
  })

  test.each([
    ['a: 1\n\n{b: 2}: 3'],
    ['a: 1\n# hi\n{b: 2}: 3']
  ])('flow map key %j', src => {
    expect(js(src)).toEqual(
      new Map<unknown, unknown>([
        ['a', 1],
        [new Map([['b', 2]]), 3]
      ])
    )
  })

  test('entries after the flow key are kept', () => {
    expect(js('a: 1\n\n\n[x, y]: z\nb: 2')).toEqual(
      new Map<unknown, unknown>([
        ['a', 1],
        [['x', 'y'], 'z'],
        ['b', 2]
      ])
    )
  })

  test('consecutive flow keys', () => {
    expect(js('[a]: 1\n\n[b]: 2\n# c\n[c]: 3')).toEqual(
      new Map<unknown, unknown>([
        [['a'], 1],
        [['b'], 2],
        [['c'], 3]
      ])
    )
  })

  test('nested block map', () => {
    expect(js('top:\n  a: 1\n\n  [x]: y\n  b: 2')).toEqual(
      new Map<unknown, unknown>([
        [
          'top',
          new Map<unknown, unknown>([
            ['a', 1],
            [['x'], 'y'],
            ['b', 2]
          ])
        ]
      ])
    )
    expect(js('top:\n  a: 1\n  # c\n  [x]: y\nafter: 3')).toEqual(
      new Map<unknown, unknown>([
        [
          'top',
          new Map<unknown, unknown>([
            ['a', 1],
            [['x'], 'y']
          ])
        ],
        ['after', 3]
      ])
    )
  })

  test('after an explicit key with a value', () => {
    expect(js('? a\n: 1\n\n[3]: 4')).toEqual(
      new Map<unknown, unknown>([
        ['a', 1],
        [[3], 4]
      ])
    )
  })

  test('comments and blank lines stay where they were', () => {
    const src = 'top:\n  a: 1\n  # note\n\n  [x]: y\n  b: 2\n'
    const out = String(YAML.parseDocument(src))
    expect(out).toContain('  a: 1\n  # note\n\n')
    expect(out.indexOf('# note')).toBeLessThan(out.indexOf('[ x ]'))
    expect(YAML.parse(out)).toEqual(YAML.parse(src))
  })

  test('a comment belongs to the flow key after it', () => {
    const doc = YAML.parseDocument<YAML.YAMLMap, false>('a: 1\n# c\n[x]: 2\n# d\n[y]: 3\n')
    expect(doc.errors).toHaveLength(0)
    expect(doc.value.comment).toBeFalsy()
    const keys = [...doc.value.pairs()].map(pair => (pair.key as YAML.Node).commentBefore ?? null)
    expect(keys).toEqual([null, ' c', ' d'])
    expect(String(doc)).toBe('a: 1\n# c\n? [ x ]\n: 2\n# d\n? [ y ]\n: 3\n')
  })
})

describe('hidden: unchanged behaviour', () => {
  test('flow key after an empty explicit key', () => {
    const doc = YAML.parseDocument('?\n[3]: 4')
    expect(doc.errors).toHaveLength(0)
    expect(doc.value).toMatchObject(
      _map([
        [null, { key: { value: null }, value: null }],
        [_seq(3), 4]
      ])
    )
  })

  test.each([
    ['1: 2\n[3]: 4', [[1, 2], [[3], 4]]],
    ['\n[3]: 4', [[[3], 4]]],
    ['# c\n[3]: 4\n5: 6', [[[3], 4], [5, 6]]],
    ['a:\n\n[3]: 4', [['a', null], [[3], 4]]]
  ] as const)('%j', (src, entries) => {
    expect(js(src)).toEqual(new Map(entries as any))
  })

  test('block sequence of maps', () => {
    expect(YAML.parse('- a: 1\n\n  b: 2\n-\n  c: 3')).toEqual([{ a: 1, b: 2 }, { c: 3 }])
  })
})
