package postgres

// Query paginasi cursor untuk daftar yang diurutkan DESC (produk, supplier,
// kategori, purchase order) memakai pola berikut:
//
//	Next: WHERE id < cursor_id ORDER BY id DESC LIMIT n
//	Prev: WHERE id > cursor_id ORDER BY id ASC  LIMIT n
//
// Query "Prev" sengaja diurutkan ASC: untuk mengambil n baris tepat di atas
// cursor, arah pengurutan harus berlawanan dengan "Next". Akibatnya hasil "Prev"
// tiba dalam urutan terbalik dan WAJIB dibalik lagi sebelum dikirim ke klien.
// Tanpa itu, halaman terakhir tampil dengan urutan terbalik dan isinya
// meloncat-loncat, misalnya halaman 3 [2,1] lalu mundur menghasilkan [4]
// alih-alih [4,3].
//
// Catatan:
//   - Daftar yang diurutkan ASC (roles) memakai pola kebalikan dan tidak
//     perlu dibalik.
//   - Cursor untuk "Prev" harus berupa id PERTAMA dari halaman yang sedang
//     ditampilkan, bukan id terakhir dari halaman sebelumnya.
func reverseRows[T any](rows []T) {
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
}
