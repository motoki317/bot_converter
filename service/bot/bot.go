package bot

import (
	"context"
	"log"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/traPtitech/go-traq"
	traqbot "github.com/traPtitech/traq-ws-bot"
	"github.com/traPtitech/traq-ws-bot/payload"

	"git.trap.jp/toki/bot_converter/repository"
)

type command struct {
	names  []string
	handle func(h *Handlers, e *messageCreatedEvent, args []string) error
}

type Handlers struct {
	repo     repository.Repository
	api      *traq.APIClient
	botID    uuid.UUID
	prefix   string
	origin   string
	commands map[string]*command
}

type Config struct {
	TraqOrigin  string
	AccessToken string
	BotID       uuid.UUID
	Prefix      string
	Origin      string
}

// NewBotService creates a new bot service.
func NewBotService(c Config, repo repository.Repository) (*traqbot.Bot, error) {
	b, err := traqbot.NewBot(&traqbot.Options{
		Origin:      c.TraqOrigin,
		AccessToken: c.AccessToken,
	})
	if err != nil {
		return nil, err
	}

	h := &Handlers{
		repo:     repo,
		api:      b.API(),
		botID:    c.BotID,
		prefix:   c.Prefix,
		origin:   c.Origin,
		commands: make(map[string]*command),
	}
	h.setUpCommands()
	h.setUpHandlers(b)

	return b, nil
}

func (h *Handlers) setUpCommands() {
	commands := []*command{
		ping(),
		create(),
		config(),
		info(),
		list(),
		deleteConverter(),
	}
	commands = append(commands, help(commands))

	for _, c := range commands {
		for _, name := range c.names {
			if _, ok := h.commands[name]; ok {
				log.Fatalf("command name conflict: " + name)
			}
			h.commands[name] = c
		}
	}
}

func (h *Handlers) setUpHandlers(b *traqbot.Bot) {
	b.OnMessageCreated(func(p *payload.MessageCreated) {
		h.handleMessageCreated(&messageCreatedEvent{
			Base:    p.Base,
			Message: p.Message,
			IsDM:    false,
		})
	})
	b.OnDirectMessageCreated(func(p *payload.DirectMessageCreated) {
		h.handleMessageCreated(&messageCreatedEvent{
			Base:    p.Base,
			Message: p.Message,
			IsDM:    true,
		})
	})
}

type messageCreatedEvent struct {
	payload.Base
	Message payload.Message
	IsDM    bool
}

func (h *Handlers) handleMessageCreated(e *messageCreatedEvent) {
	// Do not process own message
	if e.Message.User.ID == h.botID.String() {
		return
	}
	// Do not process bot messages
	if e.Message.User.Bot {
		return
	}

	args := strings.Fields(e.Message.PlainText)

	for i, arg := range args {
		if strings.HasPrefix(arg, h.prefix) {
			cmdName := arg[len(h.prefix):]
			if c, ok := h.commands[cmdName]; ok {
				args[i] = cmdName
				// e.g. PlainText of "@BOT_example /ping arg1  arg2  " will be handed to command as
				// []string{"ping", "arg1", "arg2"}
				err := c.handle(h, e, args[i:])
				if err != nil {
					log.Printf("an error occurred while handling user command: %s\n", err)
				}
				return
			}
		}
	}
}

// getChannelPath gets the path to the channel.
func (h *Handlers) getChannelPath(channelID string) (string, error) {
	c, _, err := h.api.ChannelApi.GetChannel(context.TODO(), channelID).Execute()
	if err != nil {
		return "", err
	}
	if !c.ParentId.IsSet() || c.ParentId.Get() == nil {
		return c.Name, nil
	}
	p, err := h.getChannelPath(*c.ParentId.Get())
	if err != nil {
		return "", err
	}
	return p + "/" + c.Name, nil
}

// postMessage posts message to the channel in which the event happened.
func (h *Handlers) postMessage(e *messageCreatedEvent, message string) (*traq.Message, error) {
	embed := true
	m, _, err := h.api.ChannelApi.PostMessage(context.TODO(), e.Message.ChannelID).PostMessageRequest(traq.PostMessageRequest{
		Content: message,
		Embed:   &embed,
	}).Execute()
	return m, err
}

// postDirectMessage posts message to the specified user.
func (h *Handlers) postDirectMessage(userID string, message string) (*traq.Message, error) {
	embed := true
	m, _, err := h.api.UserApi.PostDirectMessage(context.TODO(), userID).PostMessageRequest(traq.PostMessageRequest{
		Content: message,
		Embed:   &embed,
	}).Execute()
	return m, err
}
