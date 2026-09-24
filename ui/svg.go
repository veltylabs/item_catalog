//go:build !wasm

package ui

import (
	"webtyp.com/svg"
	"webtyp.com/svg/sprite"
)

// Icons es un tipo que aloja IconID/IconSvg para su descubrimiento y uso en config.
type Icons struct{}

func (m *Icons) IconSvg() *sprite.Sprite {
	return sprite.NewSprite(
		sprite.Define(svg.Icon("catalog_item"), "0 0 16 16",
			sprite.Path("M2 2h5v5H2zm7 0h5v5H9zm-7 7h5v5H2zm7 0h5v5H9z"),
		),
	)
}
