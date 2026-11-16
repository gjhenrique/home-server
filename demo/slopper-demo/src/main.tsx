import React from 'react';
import { createRoot } from 'react-dom/client';
import { download, identity, infer, insert, list, upload } from './helper';
import './style.css';

function App() {
  const [message, setMessage] = React.useState('Choose an action to see its response.');
  const [user, setUser] = React.useState('');
  const [fileName, setFileName] = React.useState('');
  const authDomain = location.hostname.split('.').slice(-3).join('.');

  async function run(action: () => Promise<unknown>) {
    try { setMessage(JSON.stringify(await action(), null, 2)); }
    catch (error) { setMessage(String(error)); }
  }

  async function chooseFile(file?: File) {
    if (!file) return;
    setFileName(file.name);
    await run(() => upload('files', file));
  }

  return (
    <main>
      <header>
        <span className="mark">S</span>
        <div><h1>Slopper</h1><p>Private app hosting and backend</p></div>
        <a href={`https://auth.${authDomain}/logout`}>Log out</a>
      </header>
      <section>
        <h2>Session</h2>
        <button onClick={() => run(async () => { const current = await identity(); setUser(current.email); return current; })}>Check identity</button>
        {user && <span>{user}</span>}
      </section>
      <section>
        <h2>Database</h2>
        <p>Write and read a row in the demo table.</p>
        <button onClick={() => run(() => insert('demo', { message: 'Hello from Slopper', created_at: new Date().toISOString() }))}>Write row</button>
        <button onClick={() => run(() => list('demo'))}>Read rows</button>
      </section>
      <section>
        <h2>Private files</h2>
        <p>Upload a file, then download the same object.</p>
        <input type="file" onChange={(event) => void chooseFile(event.target.files?.[0])} />
        <button disabled={!fileName} onClick={() => run(async () => {
          const blob = await download('files', fileName);
          return { name: fileName, bytes: blob.size, type: blob.type };
        })}>Download uploaded file</button>
      </section>
      <section>
        <h2>AI</h2>
        <p>Uses the model selected by the Slopper administrator.</p>
        <button onClick={() => run(() => infer([{ role: 'user', content: 'Say hello in one sentence.' }]))}>Ask the configured model</button>
      </section>
      <pre>{message}</pre>
    </main>
  );
}

createRoot(document.getElementById('root')!).render(<App />);
