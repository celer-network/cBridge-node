CREATE DATABASE IF NOT EXISTS cbridge;
CREATE USER IF NOT EXISTS cbridge;
GRANT ALL ON DATABASE cbridge TO cbridge;
SET DATABASE TO cbridge;

-- monitored on-chain events
CREATE TABLE IF NOT EXISTS monitor (
    event TEXT PRIMARY KEY NOT NULL,
    blocknum INT NOT NULL,
    blockidx INT NOT NULL,
    restart BOOL NOT NULL
);

CREATE TABLE IF NOT EXISTS transfer (
    tid TEXT PRIMARY KEY NOT NULL,
    txhash TEXT NOT NULL DEFAULT '',
    chainid INT NOT NULL,
    token TEXT NOT NULL,
    transfertype INT NOT NULL,
    timelock TIMESTAMPTZ NOT NULL,
    hashlock TEXT NOT NULL,
    status INT NOT NULL,
    relatedtid TEXT NOT NULL,
    relatedchainid INT NOT NULL,
    relatedtoken TEXT NOT NULL,
    amount TEXT NOT NULL,
    fee TEXT NOT NULL,
    transfergascost TEXT,
    confirmgascost TEXT,
    refundgascost TEXT,
    preimage TEXT,
    senderaddr TEXT NOT NULL,
    receiveraddr TEXT NOT NULL,
    txconfirmhash TEXT NOT NULL DEFAULT '',
    txrefundhash TEXT NOT NULL DEFAULT '',
    updatets TIMESTAMPTZ NOT NULL,
    createts TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS transfer_create_ts_idx ON transfer (createts);
CREATE INDEX IF NOT EXISTS transfer_chain_id_idx ON transfer (chainid);
