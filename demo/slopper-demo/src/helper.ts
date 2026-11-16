const base = '/_slopper';
let csrfToken = '';
async function request(path: string, init: RequestInit = {}) { const headers = new Headers(init.headers); if (csrfToken) headers.set('X-Slopper-CSRF', csrfToken); const response = await fetch(base + path, {...init, headers, credentials:'same-origin'}); if (response.status === 401) { if (init.method && init.method !== 'GET') throw new Error('Session expired; reload after signing in.'); location.assign('/oauth2/google'); throw new Error('Signing in'); } if (!response.ok) throw new Error(await response.text()); return response; }
export async function identity(){const r=await request('/identity');const value=await r.json();csrfToken=value.csrf;return value;}
export async function list(table:string){return (await request(`/db/${encodeURIComponent(table)}`)).json();}
export async function insert(table:string,row:unknown){return (await request(`/db/${encodeURIComponent(table)}`,{method:'POST',headers:{'Content-Type':'application/json','Prefer':'return=representation'},body:JSON.stringify(row)})).json();}
export async function upload(bucket:string,file:File){return request(`/objects/${encodeURIComponent(bucket)}/${encodeURIComponent(file.name)}`,{method:'PUT',headers:{'Content-Type':file.type||'application/octet-stream'},body:file});}
export async function download(bucket:string,name:string){return (await request(`/objects/${encodeURIComponent(bucket)}/${encodeURIComponent(name)}`)).blob();}
export async function infer(messages:unknown[]){return (await request('/ai/chat/completion',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({messages})})).json();}
