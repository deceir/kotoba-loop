import {describe,expect,it} from 'vitest';import {normalizeKana} from './kana'
describe('normalizeKana',()=>{it('converts romaji',()=>expect(normalizeKana('tomodachi')).toBe('ともだち'));it('normalizes spaces and case',()=>expect(normalizeKana(' OI SHII ')).toBe('おいしい'))})
