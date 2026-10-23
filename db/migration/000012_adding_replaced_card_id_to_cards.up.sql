ALTER TABLE cards
ADD COLUMN replaced_card_id  BIGINT REFERENCES cards(id) ;