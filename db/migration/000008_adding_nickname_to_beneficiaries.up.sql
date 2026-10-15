ALTER TABLE beneficiaries
    ADD COLUMN nickname text ,
    ADD COLUMN deleted_at timestamptz ;
