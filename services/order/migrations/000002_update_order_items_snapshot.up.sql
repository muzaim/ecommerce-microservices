-- Hapus tabel cached_products
DROP TABLE IF EXISTS cached_products;

-- Tambahkan kolom product_name di order_items
ALTER TABLE order_items ADD COLUMN IF NOT EXISTS product_name VARCHAR(255) NOT NULL DEFAULT '';
