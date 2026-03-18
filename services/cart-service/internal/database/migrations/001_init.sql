CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS carts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS cart_items (
    id SERIAL PRIMARY KEY,
    cart_id INTEGER NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    product_id INTEGER NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (cart_id, product_id)
);

DO $$
BEGIN
  IF to_regclass('public.carts') IS NOT NULL THEN
    EXECUTE 'DROP TRIGGER IF EXISTS set_timestamp_carts ON carts';
    EXECUTE 'CREATE TRIGGER set_timestamp_carts BEFORE UPDATE ON carts FOR EACH ROW EXECUTE PROCEDURE trigger_set_timestamp()';
  END IF;

  IF to_regclass('public.cart_items') IS NOT NULL THEN
    EXECUTE 'DROP TRIGGER IF EXISTS set_timestamp_cart_items ON cart_items';
    EXECUTE 'CREATE TRIGGER set_timestamp_cart_items BEFORE UPDATE ON cart_items FOR EACH ROW EXECUTE PROCEDURE trigger_set_timestamp()';
  END IF;
END
$$;

