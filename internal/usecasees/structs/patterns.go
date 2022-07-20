package structs

func PatternBaseBUY(deltaMAX, deltaLastMIN float64) bool {
	if deltaMAX > 30 && deltaLastMIN > 30 {
		return true
	}

	return false
}

// https://blog.roboforex.com/ru/blog/2019/09/06/svechnoj-analiz-na-foreks-osnovnye-principy-varianty-primenenija/

func PatternShootingStar(deltaLastMIN, deltaMAX float64) bool {
	// Падающая звезда, SELL

	if deltaLastMIN < 10 && deltaMAX > 70 {
		return true
	}

	return false
}

func PatternGallows(deltaLastMIN, deltaMAX float64) bool {
	//Висельник SELL

	if deltaLastMIN > 70 && deltaMAX < 10 {
		return true
	}

	return false
}
