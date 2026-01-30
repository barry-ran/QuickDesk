-- Create devices table
CREATE TABLE IF NOT EXISTS devices (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(9) UNIQUE NOT NULL,
    device_uuid UUID UNIQUE NOT NULL,
    os VARCHAR(50),
    os_version VARCHAR(50),
    app_version VARCHAR(20),
    online BOOLEAN DEFAULT false,
    last_seen TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_device_id ON devices(device_id);
CREATE INDEX IF NOT EXISTS idx_device_uuid ON devices(device_uuid);
CREATE INDEX IF NOT EXISTS idx_online ON devices(online);
CREATE INDEX IF NOT EXISTS idx_last_seen ON devices(last_seen);

-- Comments
COMMENT ON TABLE devices IS 'QuickDesk host devices';
COMMENT ON COLUMN devices.device_id IS '9-digit unique device identifier';
COMMENT ON COLUMN devices.device_uuid IS 'UUID for internal use';
COMMENT ON COLUMN devices.online IS 'Whether the device is currently online';
COMMENT ON COLUMN devices.last_seen IS 'Last time the device was seen online';
