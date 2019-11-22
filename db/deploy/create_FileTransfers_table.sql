
DROP TABLE IF EXISTS TransferRecord;
CREATE TABLE TransferRecord (
    `id` int AUTO_INCREMENT  PRIMARY KEY,
    `local_file_name` VARCHAR(254) NOT NULL COMMENT 'The name of the file on disk',
    `local_file_path` TEXT NOT NULL COMMENT 'Absolute path to the file on the disk',
    `local_file_size` INT COMMENT 'On Disk File size',
    `remote_file_name` TExT COMMENT 'Name of the Remote file',
    `remote_file_path` TEXT NOT NULL COMMENT 'Path to the remote file',
    `remote_file_size` INT COMMENT 'Remote File size',
    `recipient_name`   TEXT COMMENT 'Natural language name for the recipient ie. Bank, Customer',
    `sender_name`      TEXT COMMENT 'Name of the sender',
    `local_file_hash`  VARCHAR(254) COMMENT 'Local File Hash',
    `transferred_file_hash` VARCHAR(254) COMMENT 'Hash of the transferred bytes',
    `local_host_id`    TEXT NOT NULL COMMENT 'Host identifier of the local system',
    `remote_host`      TEXT NOT NULL COMMENT 'Host identifier of the remote system',
    `transfer_start`   DATETIME NOT NULL COMMENT 'Date and time transfer process started',
    `transfer_end`     DATETIME COMMENT 'Date and time transfer process started',
    `transfer_errors`  TEXT COMMENT 'Transfer Errors',
    `correlation_id`    VARCHAR(254) COMMENT 'CorrelationId',
    `created_at`       DATETIME NOT NULL COMMENT "Date record was added",
    `updated_at`       DATETIME COMMENT "Date record was updated",
    `deleted_at`       DATETIME COMMENT "Date record was remoted",
    
    UNIQUE INDEX file_correlationID USING HASH (local_file_name,correlation_id)

) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_bin;
