import * as YAML from 'yaml'
import { _map, _seq } from '../_utils.ts'

describe('flow collection key after a blank line', () => {
  test('flow sequence key', () => {
    const doc = YAML.parseDocument('1: 2\n\n[3]: 4')
    expect(doc.errors).toHaveLength(0)
    expect(doc.value).toMatchObject(
      _map([
        [1, 2],
        [_seq(3), 4]
      ])
    )
  })
})
