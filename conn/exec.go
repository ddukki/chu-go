package conn

import (
	"context"

	"github.com/ClickHouse/ch-go/proto"
)

func (c *Conn) Exec(ctx context.Context, query string) error {
	if err := c.lock(); err != nil {
		return err
	}
	defer c.unlock()

	q := proto.Query{
		Body:        query,
		Stage:       proto.StageComplete,
		Compression: c.cfg.Compression,
		Info:        makeClientInfo(c.server),
		Settings:    c.cfg.Settings,
	}
	c.writer.ChainBuffer(func(b *proto.Buffer) {
		q.EncodeAware(b, c.server.Revision)
	})
	if _, err := c.writer.Flush(); err != nil {
		return &Error{Kind: KindNetwork, Message: "flush query", Err: err}
	}

	if err := c.sendEmptyBlock(); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			c.sendCancel()
			return &Error{Kind: KindInternal, Message: "context canceled", Err: ctx.Err()}
		default:
		}

		code, err := c.reader.UVarInt()
		if err != nil {
			return &Error{Kind: KindNetwork, Message: "read server code", Err: err}
		}

		switch proto.ServerCode(code) {
		case proto.ServerCodeData, proto.ServerCodeTotals, proto.ServerCodeExtremes:
			if err := c.skipBlock(); err != nil {
				return err
			}
		case proto.ServerCodeEndOfStream:
			return nil
		case proto.ServerCodeException:
			var ex proto.Exception
			if err := ex.DecodeAware(c.reader, proto.Version); err != nil {
				return &Error{Kind: KindProtocol, Message: "decode exception", Err: err}
			}
			return &Error{Kind: KindServer, Message: ex.Message, ServerCode: int(ex.Code)}
		case proto.ServerCodeProgress:
			var p proto.Progress
			if err := p.DecodeAware(c.reader, c.server.Revision); err != nil {
				return &Error{Kind: KindProtocol, Message: "decode progress", Err: err}
			}
			if c.OnProgress != nil {
				c.OnProgress(p)
			}
		case proto.ServerCodeProfile:
			var p proto.Profile
			if err := p.DecodeAware(c.reader, c.server.Revision); err != nil {
				return &Error{Kind: KindProtocol, Message: "decode profile", Err: err}
			}
			if c.OnProfile != nil {
				c.OnProfile(p)
			}
		case proto.ServerProfileEvents:
			if err := c.skipBlock(); err != nil {
				return err
			}
		case proto.ServerCodeLog:
			if err := c.skipBlock(); err != nil {
				return err
			}
		default:
		}
	}
}

func (c *Conn) Ping(ctx context.Context) error {
	if err := c.lock(); err != nil {
		return err
	}
	defer c.unlock()

	c.writer.ChainBuffer(func(b *proto.Buffer) {
		b.PutUVarInt(uint64(proto.ClientCodePing))
	})
	if _, err := c.writer.Flush(); err != nil {
		return &Error{Kind: KindNetwork, Message: "flush ping", Err: err}
	}

	code, err := c.reader.UVarInt()
	if err != nil {
		return &Error{Kind: KindNetwork, Message: "read pong", Err: err}
	}
	if proto.ServerCode(code) != proto.ServerCodePong {
		return &Error{Kind: KindProtocol, Message: "unexpected ping response"}
	}
	return nil
}

func (c *Conn) sendEmptyBlock() error {
	c.writer.ChainBuffer(func(b *proto.Buffer) {
		b.PutUVarInt(uint64(proto.ClientCodeData))
		var block proto.Block
		block.EncodeAware(b, c.server.Revision)
	})
	if _, err := c.writer.Flush(); err != nil {
		return &Error{Kind: KindNetwork, Message: "flush empty block", Err: err}
	}
	return nil
}

func (c *Conn) sendCancel() {
	c.writer.ChainBuffer(func(b *proto.Buffer) {
		b.PutUVarInt(uint64(proto.ClientCodeCancel))
	})
	c.writer.Flush()
}

func (c *Conn) skipBlock() error {
	var results proto.Results
	var block proto.Block
	if err := block.DecodeBlock(c.reader, c.server.Revision, results.Auto()); err != nil {
		return &Error{Kind: KindProtocol, Message: "skip block", Err: err}
	}
	return nil
}
