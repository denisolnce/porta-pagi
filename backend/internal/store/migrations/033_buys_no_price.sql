-- The Market's wishlist asks in one field, a few words or a link (owner,
-- 2026-11-15). A limit a house already typed joins its words before its
-- column goes, so nothing a villager wrote is lost.
UPDATE buys SET text = trim(text || ' ≤ ' || max_price) WHERE max_price <> '';
ALTER TABLE buys DROP COLUMN max_price;
