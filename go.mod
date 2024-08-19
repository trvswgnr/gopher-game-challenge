module game

go 1.19

require game/engine v0.0.0-00010101000000-000000000000

require github.com/veandco/go-sdl2 v0.4.40 // indirect

replace game/engine => ./engine
