-- Fix dedup constraint: change from single source_ref_id UNIQUE to composite UNIQUE(source, source_ref_id)
ALTER TABLE candidate_applications DROP CONSTRAINT candidate_applications_source_ref_id_key;
ALTER TABLE candidate_applications ADD CONSTRAINT candidate_applications_source_ref_id_key UNIQUE(source, source_ref_id);
