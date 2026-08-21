package script

func Clear2Map(m map[string]map[string]string) {
	// 1. Duyệt qua tầng 1 để tìm các map con ở tầng 2
	for key1, subMap := range m {
		// 2. Nạo vét sạch ruột của con map tầng 2 (Giữ lại khung RAM của tầng 2)
		clear(subMap)

		// 3. Xóa key ở tầng 1 (Giữ lại khung RAM của tầng 1)
		delete(m, key1)
	}
}

func Clear3Map(m map[string]map[string]map[int]string) {
	for key1, map2D := range m {
		for key2, map1D := range map2D {
			// Nạo vét tầng 3 (Tầng sâu nhất)
			clear(map1D)
			delete(map2D, key2)
		}
		delete(m, key1)
	}
}
