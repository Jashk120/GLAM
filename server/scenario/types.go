package scenario

import (
	"encoding/json"
	"fmt"
)

// Position represents x,y grid position.
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Size struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}

type Outcome struct {
	Stat  *string `json:"stat,omitempty"`
	Delta *int    `json:"delta,omitempty"`
	Toast *string `json:"toast,omitempty"`
}

type Region struct {
	ID     string  `json:"id"`
	Name   *string `json:"name,omitempty"`
	X      *int    `json:"x,omitempty"`
	Y      *int    `json:"y,omitempty"`
	Width  *int    `json:"width,omitempty"`
	Height *int    `json:"height,omitempty"`
	Type   *string `json:"type,omitempty"`
}

type World struct {
	Template string   `json:"template"`
	Spawn    Position `json:"spawn"`
	Size     Size     `json:"size"`
	Theme    *string  `json:"theme,omitempty"`
	Regions  []Region `json:"regions,omitempty"`
}

type Appearance struct {
	SpriteID *string `json:"spriteId,omitempty"`
	Color    *string `json:"color,omitempty"`
}

// Interaction variants

type InteractionDialogue struct {
	Type      string   `json:"type"`
	Text      string   `json:"text"`
	Speaker   *string  `json:"speaker,omitempty"`
	Cooldown  *int     `json:"cooldown,omitempty"`
	Auto      *bool    `json:"auto,omitempty"`
	OnCorrect *Outcome `json:"onCorrect,omitempty"`
	OnWrong   *Outcome `json:"onWrong,omitempty"`
}

type MCQOption struct {
	Text        string  `json:"text"`
	Correct     bool    `json:"correct"`
	Explanation *string `json:"explanation,omitempty"`
}

type InteractionMCQ struct {
	Type      string      `json:"type"`
	Question  string      `json:"question"`
	Options   []MCQOption `json:"options"`
	AllowRetry *bool      `json:"allowRetry,omitempty"`
	Cooldown  *int        `json:"cooldown,omitempty"`
	Auto      *bool       `json:"auto,omitempty"`
	OnCorrect *Outcome    `json:"onCorrect,omitempty"`
	OnWrong   *Outcome    `json:"onWrong,omitempty"`
}

type InteractionMath struct {
	Type      string      `json:"type"`
	Question  string      `json:"question"`
	Answer    interface{} `json:"answer"`
	Tolerance *float64    `json:"tolerance,omitempty"`
	Hint      *string     `json:"hint,omitempty"`
	Cooldown  *int        `json:"cooldown,omitempty"`
	Auto      *bool       `json:"auto,omitempty"`
	OnCorrect *Outcome    `json:"onCorrect,omitempty"`
	OnWrong   *Outcome    `json:"onWrong,omitempty"`
}

type ShopItem struct {
	ID          *string `json:"id,omitempty"`
	Name        string  `json:"name"`
	Price       int     `json:"price"`
	Icon        *string `json:"icon,omitempty"`
	Description *string `json:"description,omitempty"`
}

type InteractionShop struct {
	Type      string     `json:"type"`
	Currency  *string    `json:"currency,omitempty"`
	Items     []ShopItem `json:"items"`
	Cooldown  *int       `json:"cooldown,omitempty"`
	Auto      *bool      `json:"auto,omitempty"`
	OnCorrect *Outcome   `json:"onCorrect,omitempty"`
	OnWrong   *Outcome   `json:"onWrong,omitempty"`
}

type InteractionInformation struct {
	Type      string   `json:"type"`
	Title     *string  `json:"title,omitempty"`
	Content   string   `json:"content"`
	Image     *string  `json:"image,omitempty"`
	Cooldown  *int     `json:"cooldown,omitempty"`
	Auto      *bool    `json:"auto,omitempty"`
	OnCorrect *Outcome `json:"onCorrect,omitempty"`
	OnWrong   *Outcome `json:"onWrong,omitempty"`
}

// Interaction is a discriminated union of 5 variants.
type Interaction struct {
	Type string
	Dialogue    *InteractionDialogue
	MCQ         *InteractionMCQ
	Math        *InteractionMath
	Shop        *InteractionShop
	Information *InteractionInformation
	Raw         json.RawMessage
}

func (i *Interaction) UnmarshalJSON(data []byte) error {
	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return err
	}
	i.Type = probe.Type
	i.Raw = json.RawMessage(data)
	switch probe.Type {
	case "dialogue":
		var v InteractionDialogue
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		i.Dialogue = &v
	case "mcq":
		var v InteractionMCQ
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		i.MCQ = &v
	case "math":
		var v InteractionMath
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		i.Math = &v
	case "shop":
		var v InteractionShop
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		i.Shop = &v
	case "information":
		var v InteractionInformation
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		i.Information = &v
	default:
		return fmt.Errorf("unknown interaction type %q", probe.Type)
	}
	return nil
}

func (i Interaction) MarshalJSON() ([]byte, error) {
	if len(i.Raw) > 0 {
		return i.Raw, nil
	}
	switch i.Type {
	case "dialogue":
		return json.Marshal(i.Dialogue)
	case "mcq":
		return json.Marshal(i.MCQ)
	case "math":
		return json.Marshal(i.Math)
	case "shop":
		return json.Marshal(i.Shop)
	case "information":
		return json.Marshal(i.Information)
	default:
		return nil, fmt.Errorf("unknown interaction type %q", i.Type)
	}
}

// Entity types

type Character struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Profession  *string      `json:"profession,omitempty"`
	Appearance  *Appearance  `json:"appearance,omitempty"`
	Position    Position     `json:"position"`
	Plot        *string      `json:"plot,omitempty"`
	Interaction *Interaction `json:"interaction,omitempty"`
}

type Building struct {
	ID          string       `json:"id"`
	TypeAssetID string       `json:"typeAssetId"`
	Position    Position     `json:"position"`
	Width       *int         `json:"width,omitempty"`
	Height      *int         `json:"height,omitempty"`
	Plot        *string      `json:"plot,omitempty"`
	Interaction *Interaction `json:"interaction,omitempty"`
}

type ObjectEntity struct {
	ID          string       `json:"id"`
	AssetID     string       `json:"assetId"`
	Position    Position     `json:"position"`
	Plot        *string      `json:"plot,omitempty"`
	Interaction *Interaction `json:"interaction,omitempty"`
}

type MissionTrigger struct {
	EntityID      *string `json:"entityId,omitempty"`
	InteractionID *string `json:"interactionId,omitempty"`
	Auto          *bool   `json:"auto,omitempty"`
}

type Mission struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Trigger     *MissionTrigger `json:"trigger,omitempty"`
	CheckAtEnd  *bool           `json:"checkAtEnd,omitempty"`
	Done        *bool           `json:"done,omitempty"`
}

type Scenario struct {
	ID         string         `json:"id"`
	Title      string         `json:"title"`
	Version    *string        `json:"version,omitempty"`
	World      World          `json:"world"`
	Characters []Character    `json:"characters"`
	Buildings  []Building     `json:"buildings"`
	Objects    []ObjectEntity `json:"objects"`
	Missions   []Mission      `json:"missions"`
}
