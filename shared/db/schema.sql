CREATE TABLE if NOT EXISTS identity (
    peer_id BLOB PRIMARY KEY,
    username TEXT NOT NULL DEFAULT 'anonymous',
    public_key BLOB NOT NULL,
    encrypted_private_key BLOB NOT NULL DEFAULT x'',
    salt BLOB NOT NULL DEFAULT x'',
    nonce BLOB NOT NULL DEFAULT x''
);