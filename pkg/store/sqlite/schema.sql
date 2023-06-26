--
-- Schema for sqlite store
--

CREATE TABLE IF NOT EXISTS modems (
    mac_address  	TEXT NOT NULL PRIMARY KEY,
    ipv6           	TEXT NOT NULL,
    switch_port		INTEGER NOT NULL,
    model       	TEXT NOT NULL,
    state          	INTEGER NOT NULL,
    firmware       	TEXT NOT NULL,
    serial         	TEXT NOT NULL,
    kernel         	TEXT,
    upgraded       	BOOLEAN NOT NULL,
    last_updated   	INTEGER,
    fail_count     	INTEGER,
    sim_provider   	TEXT,
    sim_status     	BOOLEAN,
    imei           	TEXT,
    iccid          	TEXT,
    imsi           	TEXT,
	progress 		INTEGER
);