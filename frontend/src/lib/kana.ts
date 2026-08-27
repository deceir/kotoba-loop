import {toHiragana} from 'wanakana'
export const normalizeKana=(value:string)=>toHiragana(value.trim().toLowerCase()).replace(/\s/g,'')
