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

	return p.X() >= indonesiaMinLongitude && p.X() <= indonesiaMaxLongitude &&
		p.Y() >= indonesiaMinLatitude && p.Y() <= indonesiaMaxLatitude
}
