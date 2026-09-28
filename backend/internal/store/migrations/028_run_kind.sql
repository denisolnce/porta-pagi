-- A run is a trip or an order. A trip is a car: a house drives somewhere and
-- the needs ride along. An order is a delivery: a house orders from a shop, a
-- catalogue or a supplier, and the needs join it before it closes. Same
-- shape — one house, a cut-off, the needs on it — so the same table; only the
-- words differ. The app records the order, never the money: no amount column.
-- Every run written before this was a trip.
ALTER TABLE runs ADD COLUMN kind TEXT NOT NULL DEFAULT 'trip' CHECK (kind IN ('trip', 'order'));
