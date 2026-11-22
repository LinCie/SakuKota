package postgis

// Indonesia's bounding box in degrees, edges included.
const (
	indonesiaMinLongitude = 95
	indonesiaMaxLongitude = 141
	indonesiaMinLatitude  = -11
	indonesiaMaxLatitude  = 6
)

// InIndonesia reports whether p lies inside Indonesia's bounding box. A nil
// point is not inside it.
func InIndonesia(p *Point) bool {
	if p == nil {
		return false
	}

	return inIndonesia(p.X(), p.Y())
}

// MultiPolygonInIndonesia reports whether every vertex of m lies inside
// Indonesia's bounding box. A nil area is not inside it.
func MultiPolygonInIndonesia(m *MultiPolygon) bool {
	if m == nil {
		return false
	}

	for _, rings := range m.polygons {
		for _, ring := range rings {
			for _, c := range ring {
				if !inIndonesia(c[0], c[1]) {
					return false
				}
			}
		}
	}

	return true
}

func inIndonesia(lng, lat float64) bool {
	return lng >= indonesiaMinLongitude && lng <= indonesiaMaxLongitude &&
		lat >= indonesiaMinLatitude && lat <= indonesiaMaxLatitude
}
