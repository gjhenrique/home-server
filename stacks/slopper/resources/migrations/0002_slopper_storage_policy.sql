-- Apply after InsForge storage has created storage.objects. Keep Slopper's
-- private buckets inaccessible to anon except through the authenticated,
-- manifest-scoped Slopper gateway.
GRANT SELECT, INSERT, UPDATE, DELETE ON storage.objects TO anon;

CREATE POLICY slopper_anon_bucket_access ON storage.objects
  FOR ALL TO anon
  USING (left(bucket, 4) = 'slp_')
  WITH CHECK (left(bucket, 4) = 'slp_');
