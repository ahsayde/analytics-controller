CREATE TABLE IF NOT EXISTS "events" (
	"uid"	TEXT,
	"type"	TEXT,
	"reason"	TEXT,
	"message"	TEXT,
	"action"	TEXT,
	"count"	INTEGER,
	"reportingController"	TEXT,
	"reportingInstance"	TEXT,
	"firstTimestamp"	DATETIME,
	"lastTimestamp"	DATETIME,
	"involvedObject.uid"	TEXT,
	"involvedObject.apiVersion"	TEXT,
	"involvedObject.kind"	TEXT,
	"involvedObject.name"	TEXT,
	"involvedObject.namespace"	TEXT,

	PRIMARY KEY("uid")
);