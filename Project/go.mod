module github.com/KralHa0/TTK4145_16/Project

go 1.25.5

replace Driver-go => ./Hardware

replace Network-go => ./Network

require Driver-go v0.0.0-00010101000000-000000000000

require Network-go v0.0.0
