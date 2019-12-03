
DROP TABLE IF EXISTS EncryptedFiles;
CREATE TABLE EncryptedFiles (
    `id` int AUTO_INCREMENT  PRIMARY KEY,
    `local_file_name` VARCHAR(254) NOT NULL COMMENT 'The name of the file on disk',
    `local_file_path` TEXT NOT NULL COMMENT 'Absolute path to the file on the disk',
    `local_file_size` INT COMMENT 'On Disk File size',        
    `recipient_key`   TEXT COMMENT 'Fingerprint of the recipients key',
    `signing_key`      TEXT COMMENT 'Fingerprint of the signing key',
    `local_file_hash`  VARCHAR(254) COMMENT 'Local File Hash',
    'encrypted_file_hash' VARCHAR(254) COMMENT 'Hash of the encrypted file',
    `correlation_id`    VARCHAR(254) COMMENT 'CorrelationId',
    `created_at`       DATETIME NOT NULL COMMENT "Date record was added",
    `updated_at`       DATETIME COMMENT "Date record was updated",
    `deleted_at`       DATETIME COMMENT "Date record was remoted",
    
    UNIQUE INDEX pk_local_file_hash USING HASH (local_file_hash)

) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE utf8mb4_bin;
