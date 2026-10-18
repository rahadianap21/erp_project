Jika data semakin banyak, misal 1.000.000 data, apakah filter pencarian nama & email efektif jika langsung filter ketika user mengetik? atau perlu diberikan debounce?

ya, eksekusi sql seperti itu tidak efektif, jika data semakin banyak akan sangat lambat. Sebaiknya menggunakan pagination dan debounce agar tidak membebani database.