CREATE TABLE if NOT EXISTS identity (
    peer_id BLOB PRIMARY KEY,
    username TEXT NOT NULL DEFAULT 'anonymous',
    public_key BLOB NOT NULL,
    encrypted_private_key BLOB NOT NULL DEFAULT x'',
    salt BLOB NOT NULL DEFAULT x'',
    nonce BLOB NOT NULL DEFAULT x''
);


CREATE TABLE IF NOT EXISTS settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    bluetooth INTEGER NOT NULL DEFAULT 1 CHECK (bluetooth IN (0, 1)),
    webrtc INTEGER NOT NULL DEFAULT 1 CHECK (webrtc IN (0, 1)),
    mesh_relaying INTEGER NOT NULL DEFAULT 1 CHECK (mesh_relaying IN (0, 1)),
    max_relay_hops INTEGER NOT NULL DEFAULT 5 CHECK (max_relay_hops BETWEEN 1 AND 10),
    peer_discovery INTEGER NOT NULL DEFAULT 1 CHECK (peer_discovery IN (0, 1)),
    notifications INTEGER NOT NULL DEFAULT 1 CHECK (notifications IN (0, 1)),
    incoming_requests TEXT NOT NULL DEFAULT 'ASK' CHECK (incoming_requests IN ('ASK', 'ACCEPT')),
    timestamp TEXT NOT NULL DEFAULT '24H' CHECK (timestamp IN ('24H', '12H'))
);