-- 000009_modul_akuntansi.down.sql
-- Rollback Modul Akuntansi (Fase 1)

DROP TABLE IF EXISTS t_journal_entry_lines;
DROP TABLE IF EXISTS t_journal_entries;
DROP TABLE IF EXISTS t_journal_number_counters;
DROP TABLE IF EXISTS s_accounting_settings;
DROP TABLE IF EXISTS m_accounting_periods;
DROP TABLE IF EXISTS m_chart_of_accounts;

DELETE FROM s_permissions
WHERE name IN ('accounting:read', 'accounting:create', 'accounting:update',
               'accounting:delete', 'accounting:post', 'accounting:close-period');