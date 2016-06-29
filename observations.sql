-- PREPARE DATABASE

CREATE TABLE observations
(
    uid integer PRIMARY KEY AUTOINCREMENT,
    observationtime timestamp,
    windspeed real,
    winddirection varchar (3),
    significantwaveheight real,
    dominantwaveperiod integer,
    averageperiod real,
    meanwavedirection varchar (3),
    airtemperature real,
    watertemperature real,
    rowcreated timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);