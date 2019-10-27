CREATE TABLE `FileTransfers` (
    `process_start` DATETIME NOT NULL COMMENT 'Date and time transfer process was started',
    `process_errors` TEXT COMMENT 'Any errors detected in processing the file',
    `process_end` DATETIME COMMENT 'Date and time transfer process ended',
    `file_name` TINYTEXT NOT NULL COMMENT 'Name of the file being transferred',
    `file_recipient` TINYTEXT NOT NULL COMMENT 'Place that the file is being transferred to',
    `file_key` TEXT COMMENT 'Fingerprint of the key used to encrypt the file',
    `file_sender` TINYTEXT NOT NULL COMMENT 'Name of the machine that is sending the file',
    `hash_plaintext` VARCHAR(254) NOT NULL COMMENT 'Hash of the file on disk before encryption',
    `hash_ciphertext` TEXT COMMENT 'Hash of the file on disk after encryption',
    `hash_remote` TEXT COMMENT 'Hash of the file after upload to recipient',
     PRIMARY KEY(`hash_plaintext`)
) Engine=InnoDb;

ALTER DATABASE FileTransfers CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
