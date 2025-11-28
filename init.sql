-- Create database schema for Telegram Offer Bot

-- Users table
CREATE TABLE IF NOT EXISTS users (
    telegram_id BIGINT PRIMARY KEY,
    username VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Wishlists table
CREATE TABLE IF NOT EXISTS wishlists (
    id SERIAL PRIMARY KEY,
    telegram_id BIGINT NOT NULL REFERENCES users(telegram_id) ON DELETE CASCADE,
    product_name VARCHAR(500) NOT NULL,
    target_price DECIMAL(10,2),
    discount_percentage INT,
    created_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT check_target CHECK (
        (target_price IS NOT NULL AND discount_percentage IS NULL) OR
        (target_price IS NULL AND discount_percentage IS NOT NULL)
    )
);

-- Offers table (for tracking and analytics)
CREATE TABLE IF NOT EXISTS offers (
    id SERIAL PRIMARY KEY,
    product_name VARCHAR(500) NOT NULL,
    price DECIMAL(10,2),
    original_price DECIMAL(10,2),
    discount_percentage INT,
    cashback_percentage INT,
    source VARCHAR(255),
    received_at TIMESTAMP DEFAULT NOW()
);

-- Notifications table (for tracking sent notifications)
CREATE TABLE IF NOT EXISTS notifications (
    id SERIAL PRIMARY KEY,
    telegram_id BIGINT NOT NULL REFERENCES users(telegram_id) ON DELETE CASCADE,
    wishlist_id INT REFERENCES wishlists(id) ON DELETE SET NULL,
    offer_id INT REFERENCES offers(id) ON DELETE SET NULL,
    sent_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_wishlists_telegram_id ON wishlists(telegram_id);
CREATE INDEX IF NOT EXISTS idx_offers_product_name ON offers(product_name);
CREATE INDEX IF NOT EXISTS idx_notifications_telegram_id ON notifications(telegram_id);
CREATE INDEX IF NOT EXISTS idx_notifications_sent_at ON notifications(sent_at);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger for users table
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Message templates table (for web dashboard)
CREATE TABLE IF NOT EXISTS message_templates (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    product_model VARCHAR(255) NOT NULL,
    
    -- Structured fields for offer data
    title_field VARCHAR(100) NOT NULL,              -- Campo para título do produto/oferta
    description_field VARCHAR(100),                  -- Campo para descrição
    price_field VARCHAR(100) NOT NULL,               -- Campo para preço
    discount_field VARCHAR(100),                     -- Campo para desconto (opcional)
    details_fields TEXT,                             -- Campos concatenados para busca (JSON array)
    
    -- Original schema for backward compatibility
    message_schema JSONB NOT NULL,
    
    sns_topic_arn VARCHAR(500),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create trigger for message_templates table
CREATE TRIGGER update_message_templates_updated_at BEFORE UPDATE ON message_templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create index for message templates
CREATE INDEX IF NOT EXISTS idx_message_templates_active ON message_templates(is_active);
CREATE INDEX IF NOT EXISTS idx_message_templates_product_model ON message_templates(product_model);

-- Import templates table (for S3 JSON imports)
CREATE TABLE IF NOT EXISTS import_templates (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    s3_url TEXT NOT NULL,
    mapping_schema JSONB NOT NULL,  -- Maps Offer fields to JSON paths, e.g. {"ProductName": "$.titulo", "Price": "$.price"}
    is_active BOOLEAN DEFAULT true,
    last_run_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create trigger for import_templates table
CREATE TRIGGER update_import_templates_updated_at BEFORE UPDATE ON import_templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create index for import templates
CREATE INDEX IF NOT EXISTS idx_import_templates_active ON import_templates(is_active);

