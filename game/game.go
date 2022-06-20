package game

import (
	"crypto/rsa"
	"github.com/Tnze/go-mc/net"
	"github.com/Tnze/go-mc/server"
	"github.com/go-mc/server/client"
	"github.com/go-mc/server/player"
	"github.com/go-mc/server/world"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"path/filepath"
)

type Game struct {
	log *zap.Logger

	config Config

	playerProvider player.Provider
	overworld      *world.World

	keepAlive  server.KeepAlive
	playerList *server.PlayerList // playerList for updating Ping&List info
}

func NewGame(log *zap.Logger, config Config, playerList *server.PlayerList) *Game {
	overworld := world.NewProvider(filepath.Join(".", config.LevelName, "region"))

	return &Game{
		log:            log,
		config:         config,
		playerProvider: player.NewProvider(filepath.Join(".", config.LevelName, "playerdata")),
		overworld:      world.New(log.Named("overworld"), overworld),
	}
}

// AcceptPlayer 在新玩家登入时在单独的goroutine中被调用
func (g *Game) AcceptPlayer(name string, id uuid.UUID, profilePubKey *rsa.PublicKey, protocol int32, conn *net.Conn) {
	logger := g.log.With(
		zap.String("name", name),
		zap.String("uuid", id.String()),
		zap.Int32("protocol", protocol),
	)
	logger.Info("Player join")
	defer logger.Info("Player left")

	c := client.New(g.log, conn)
	p, err := g.playerProvider.GetPlayer(name, id)
	if err != nil {
		logger.Error("Read player data error", zap.Error(err))
		return
	}
	g.keepAlive.ClientJoin(c)
	defer g.keepAlive.ClientLeft(c)
	g.playerList.ClientJoin(c, server.PlayerSample{Name: name, ID: id})
	defer g.playerList.ClientLeft(c)

	if err := c.Spawn(p, g.overworld); err != nil {
		logger.Error("Spawn player error", zap.Error(err))
		return
	}
	c.Start()
}
