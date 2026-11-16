create table if not exists public.slp_slopper_demo__demo (
  id bigint generated always as identity primary key,
  message text not null,
  created_at timestamptz not null default now()
);
alter table public.slp_slopper_demo__demo enable row level security;
grant select, insert, update, delete on public.slp_slopper_demo__demo to anon;
grant usage, select on sequence public.slp_slopper_demo__demo_id_seq to anon;
create policy "Slopper demo rows" on public.slp_slopper_demo__demo
  for all to anon using (true) with check (true);
notify pgrst, 'reload schema';
