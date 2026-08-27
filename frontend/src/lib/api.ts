export type Deck={id:number;name:string;description:string;wordCount:number;addedCount:number}
export type Word={id:number;english:string;japanese:string;reading:string;deck:string}
const base=import.meta.env.VITE_API_URL||'http://localhost:8080'
export async function request<T>(path:string,options:RequestInit={},token?:string):Promise<T>{const r=await fetch(base+path,{...options,headers:{'Content-Type':'application/json',...(token?{Authorization:`Bearer ${token}`}:{})}});if(r.status===204)return null as T;const body=await r.json();if(!r.ok)throw new Error(body.error||'Request failed');return body}
