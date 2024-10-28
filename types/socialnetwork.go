package types

import (
	"bytes"
	"fmt"
	"go.dedis.ch/cs438/utils"
	"math/big"
	"sort"
	"strings"
)

// Post implements Equal
func (p Post) Equal(p2 Post) bool {
	sort.Slice(p.Comments, func(i, j int) bool {
		return p.Comments[i].CommentID < p.Comments[j].CommentID
	})

	sort.Slice(p2.Comments, func(i, j int) bool {
		return p2.Comments[i].CommentID < p2.Comments[j].CommentID
	})

	if len(p.Comments) != len(p2.Comments) {
		return false
	}

	for id := range p.Comments {
		if !p.Comments[id].Equal(p2.Comments[id]) {
			return false
		}
	}

	return p.CreatedAt.Equal(p2.CreatedAt) &&
		bytes.Equal(p.Content, p2.Content) &&
		p.PostID == p2.PostID &&
		p.Likes == p2.Likes &&
		p.AuthorID == p2.AuthorID &&
		p.UpdatedAt.Equal(p2.UpdatedAt)
}

// display Posts in a frame in the terminal (built with the help of ChatGPT)
func (p Post) String() string {
	out := new(strings.Builder)

	const boxWidth = 80

	// Function to wrap text with borders
	addBorders := func(text string) string {
		padding := boxWidth - len(text) - 3
		if padding < 0 {
			padding = 0
		}
		return "| " + text + strings.Repeat(" ", padding) + "|\n"
	}
	postIDBytes, _ := utils.DecodeChordIDFromStr(p.PostID)

	authorIDBytes, _ := utils.DecodeChordIDFromStr(p.AuthorID)

	out.WriteString("\n")
	out.WriteString("+" + strings.Repeat("-", boxWidth-2) + "+\n")
	out.WriteString(addBorders("POST"))
	out.WriteString("+" + strings.Repeat("-", boxWidth-2) + "+\n")

	// Add post details
	out.WriteString(addBorders(fmt.Sprintf("PostID:    %x", postIDBytes)))
	out.WriteString(addBorders(fmt.Sprintf("AuthorID:  %x", authorIDBytes)))
	out.WriteString(addBorders(fmt.Sprintf("Content:   %s", p.Content)))
	out.WriteString(addBorders(fmt.Sprintf("CreatedAt: %s", p.CreatedAt.Format("2006-01-02 15:04:05"))))
	out.WriteString(addBorders(fmt.Sprintf("UpdatedAt: %s", p.UpdatedAt.Format("2006-01-02 15:04:05"))))
	out.WriteString(addBorders(fmt.Sprintf("Likes:     %d", p.Likes)))
	out.WriteString(addBorders("Comments:  "))

	// Add comments with proper formatting
	for _, c := range p.Comments {
		commentText := fmt.Sprintf("- [%s] %s: %s", c.CreatedAt.Format("2006-01-02 15:04:05"), c.AuthorID, c.Content)
		out.WriteString(addBorders(commentText))
	}

	// Create a bottom border
	out.WriteString("+" + strings.Repeat("-", boxWidth-2) + "+\n")

	return out.String()
}

// Comment implements Equal
func (c Comment) Equal(c2 Comment) bool {
	return c.Content == c2.Content &&
		c.CreatedAt.Equal(c2.CreatedAt) &&
		c.AuthorID == c2.AuthorID &&
		c.PostID == c2.PostID &&
		c.CommentID == c2.CommentID
}

// -----------------------------------------------------------------------------
// FollowRequestMessage

// NewEmpty implements types.Message.
func (fr FollowRequestMessage) NewEmpty() Message {
	return &FollowRequestMessage{}
}

// Name implements types.Message.
func (fr FollowRequestMessage) Name() string {
	return "followrequest"
}

// String implements types.Message.
func (fr FollowRequestMessage) String() string {
	return "followrequeststring"
}

// HTML implements types.Message.
func (fr FollowRequestMessage) HTML() string {
	return "followrequesthtml"
}

// -----------------------------------------------------------------------------
// FollowReplyMessage

// NewEmpty implements types.Message.
func (fr FollowReplyMessage) NewEmpty() Message {
	return &FollowReplyMessage{}
}

// Name implements types.Message.
func (fr FollowReplyMessage) Name() string {
	return "followreply"
}

// String implements types.Message.
func (fr FollowReplyMessage) String() string {
	return "followreplystring"
}

// HTML implements types.Message.
func (fr FollowReplyMessage) HTML() string {
	return "followreplyhtml"
}

// -----------------------------------------------------------------------------
// BlockMessage

func (b BlockMessage) NewEmpty() Message { return &BlockMessage{} }

func (b BlockMessage) Name() string {
	return "block"
}

func (b BlockMessage) String() string {
	return "blockstring"
}

func (b BlockMessage) HTML() string {
	return "blockhtml"
}

// -----------------------------------------------------------------------------
// FlagPostMessage

// NewEmpty implements types.Message.
func (fr FlagPostMessage) NewEmpty() Message {
	return &FlagPostMessage{}
}

// Name implements types.Message.
func (fr FlagPostMessage) Name() string {
	return "flagpost"
}

// String implements types.Message.
func (fr FlagPostMessage) String() string {
	return "flagpoststring"
}

// HTML implements types.Message.
func (fr FlagPostMessage) HTML() string {
	return "flagposthtml"
}

// -----------------------------------------------------------------------------
// ChordRequestMessage

// NewEmpty implements types.Message.
func (qc ChordRequestMessage) NewEmpty() Message {
	return &ChordRequestMessage{}
}

// Name implements types.Message.
func (qc ChordRequestMessage) Name() string {
	return "ChordRequestMessage"
}

// String implements types.Message.
func (qc ChordRequestMessage) String() string {

	out := new(strings.Builder)

	// print the ChordRequestMessage in JSON format on one row
	fmt.Fprintf(out, "ChordRequestMessage{")
	fmt.Fprintf(out, "RequestID: %s,", qc.RequestID)
	fmt.Fprintf(out, "ChordNode{")
	fmt.Fprintf(out, "ID: %d, ", new(big.Int).SetBytes(qc.ChordNode.ID[:]))
	fmt.Fprintf(out, "IpAddr: %s,", qc.ChordNode.IPAddr)
	fmt.Fprintf(out, "}, ")
	fmt.Fprintf(out, "Query: %d, ", qc.Query)
	fmt.Fprintf(out, "IthFinger: %v}", qc.IthFinger)

	// print the ChordRequestMessage in JSON format on multiple rows
	//fmt.Fprintf(out, "\nQueryChordMessage{")
	//fmt.Fprintf(out, "\n\tRequestID: %s,", qc.RequestID)
	//fmt.Fprintf(out, "\n\tChordNode{")
	//fmt.Fprintf(out, "\n\t\tID: %d, ", new(big.Int).SetBytes(qc.ChordNode.ID[:]))
	//fmt.Fprintf(out, "\n\t\tIpAddr: %s,", qc.ChordNode.IPAddr)
	//fmt.Fprintf(out, "\n\t}, ")
	//fmt.Fprintf(out, "\n\tQuery: %d, ", qc.Query)
	//fmt.Fprintf(out, "\n\tIthFinger: %v}\n", qc.IthFinger)

	res := out.String()
	if res == "" {
		res = "{}"
	}

	return res
}

// HTML implements types.Message.
func (qc ChordRequestMessage) HTML() string {
	return "ChordRequestMessageHTML"
}

// -----------------------------------------------------------------------------
// ChordReplyMessage

// NewEmpty implements types.Message.
func (ac ChordReplyMessage) NewEmpty() Message {
	return &ChordReplyMessage{}
}

// Name implements types.Message.
func (ac ChordReplyMessage) Name() string {
	return "ChordReplyMessage"
}

// String implements types.Message.
func (ac ChordReplyMessage) String() string {
	return "ChordReplyMessageStr"
}

// HTML implements types.Message.
func (ac ChordReplyMessage) HTML() string {
	return "ChordReplyMessageHTML"
}

// -----------------------------------------------------------------------------

// PostRequestMessage

// NewEmpty implements types.Message.
func (pReqM PostRequestMessage) NewEmpty() Message {
	return &PostRequestMessage{}
}

// Name implements types.Message.
func (pReqM PostRequestMessage) Name() string {
	return "PostRequestMessage"
}

// String implements types.Message.
func (pReqM PostRequestMessage) String() string {
	return "PostRequestMessageStr"
}

// HTML implements types.Message.
func (pReqM PostRequestMessage) HTML() string {
	return "PostRequestMessageHTML"
}

// -----------------------------------------------------------------------------

// PostReplyMessage

// NewEmpty implements types.Message.
func (pRepM PostReplyMessage) NewEmpty() Message {
	return &PostReplyMessage{}
}

// Name implements types.Message.
func (pRepM PostReplyMessage) Name() string {
	return "PostReplyMessage"
}

// String implements types.Message.
func (pRepM PostReplyMessage) String() string {
	return "PostReplyMessageStr"
}

// HTML implements types.Message.
func (pRepM PostReplyMessage) HTML() string {
	return "PostReplyMessageHTML"
}

// -----------------------------------------------------------------------------
// CatalogRequestMessage

// NewEmpty implements types.Message.
func (GCQ CatalogRequestMessage) NewEmpty() Message { return &CatalogRequestMessage{} }

// Name implements types.Message.
func (GCQ CatalogRequestMessage) Name() string {
	return "CatalogRequestMessage"
}

// String implements types.Message.
func (GCQ CatalogRequestMessage) String() string {
	return "CatalogRequestMessageStr"
}

// HTML implements types.Message.
func (GCQ CatalogRequestMessage) HTML() string {
	return "CatalogRequestMessageHTML"
}

// -----------------------------------------------------------------------------
// CatalogReplyMessage

// NewEmpty implements types.Message.
func (GCA CatalogReplyMessage) NewEmpty() Message {
	return &CatalogReplyMessage{}
}

// Name implements types.Message.
func (GCA CatalogReplyMessage) Name() string {
	return "CatalogReplyMessage"
}

// String implements types.Message.
func (GCA CatalogReplyMessage) String() string {
	return "CatalogReplyMessageStr"
}

// HTML implements types.Message.
func (GCA CatalogReplyMessage) HTML() string {
	return "CatalogReplyMessageHTML"
}

// -----------------------------------------------------------------------------
