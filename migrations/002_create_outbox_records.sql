-- Intent: store the Outbox records: one row per committed transaction and
--   must-run handler, holding that transaction's events for the Relay.
--   idx_due serves the Relay's claim; idx_created serves Failed and Counts;
--   idx_updated serves the Purge of completed records. This is the schema of
--   contrib/event/outbox/xorm Store.Schema; change both together.
-- Compatibility: creates a new table; deploy it before code that dispatches
--   through the Outbox, which writes a row in every registration.
-- Reversal: DROP TABLE `outbox_records` after every record has completed;
--   dropping pending records loses events that have not reached their
--   handlers.
CREATE TABLE `outbox_records` (
  `id` varchar(64) NOT NULL COMMENT 'Record ID; message IDs are derived from it',
  `handler` varchar(191) NOT NULL COMMENT 'Must-run handler Name',
  `events` longtext NOT NULL COMMENT 'The transaction''s events for this handler, as JSON',
  `event_count` int NOT NULL COMMENT 'Number of events in events',
  `progress` int NOT NULL COMMENT 'Events done; only grows',
  `status` tinyint NOT NULL COMMENT 'Pending, completed, or failed',
  `attempts` int NOT NULL COMMENT 'Leases taken on the record',
  `lease_owner` varchar(191) NOT NULL COMMENT 'Relay holding the lease',
  `lease_until` bigint NOT NULL COMMENT 'Lease expiry in microseconds',
  `next_attempt_at` bigint NOT NULL COMMENT 'Next claim time in microseconds',
  `last_error` text NOT NULL COMMENT 'Error of the last failed attempt',
  `trace` text NOT NULL COMMENT 'Trace carrier of the dispatching request',
  `created_at` bigint NOT NULL COMMENT 'Commit time in microseconds',
  `updated_at` bigint NOT NULL COMMENT 'Update time in microseconds',
  PRIMARY KEY (`id`),
  KEY `idx_due` (`status`, `handler`, `next_attempt_at`),
  KEY `idx_created` (`status`, `created_at`),
  KEY `idx_updated` (`status`, `updated_at`)
) ENGINE=InnoDB
  DEFAULT CHARSET=utf8mb4
  COMMENT='Outbox records of must-run Domain Event handlers';
