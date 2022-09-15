package slack

// Field are defined as an array, and hashes contained within it will be displayed in a table inside the message attachment.
type Field struct {
	/**
	 * Shown as a bold heading above the value text. It cannot contain markup and will be escaped for you.
	 */
	Title string `json:"title"`
	/**
	 * The text value of the field. It may contain standard message markup and must be escaped as normal. May be multi-line.
	 */
	Value string `json:"value"`
	/**
	 * An optional flag indicating whether the value is short enough to be displayed side-by-side with other values.
	 */
	Short bool `json:"short,omitempty"`
}

// Attachment is Attaching content and links to messages
//
// https://api.slack.com/reference/messaging/attachments
type Attachment struct {
	/**
	 * Like traffic signals, color-coding messages can quickly communicate intent and help separate them from the flow of other messages in the timeline.
	 * An optional value that can either be one of good, warning, danger, or any hex color code (eg. #439FE0). This value is used to color the border along the left side of the message attachment.
	 */
	Color string `json:"color,omitempty"`
	/**
	 * This is optional text that appears above the message attachment block.
	 */
	Pretext string `json:"pretext,omitempty"`
	/**
	 * Small text used to display the author's name.
	 */
	AuthorName string `json:"author_name,omitempty"`
	/**
	 * A valid URL that will hyperlink the author_name text mentioned above. Will only work if author_name is present.
	 */
	AuthorLink string `json:"author_link,omitempty"`
	/**
	 * A valid URL that displays a small 16x16px image to the left of the author_name text. Will only work if author_name is present.
	 */
	AuthorIcon string `json:"author_icon,omitempty"`
	/**
	 * The title is displayed as larger, bold text near the top of a message attachment.
	 */
	Title string `json:"title,omitempty"`
	/**
	 * By passing a valid URL in the title_link parameter (optional), the title text will be hyperlinked.
	 */
	TitleLink string `json:"title_link,omitempty"`
	/**
	 * This is the main text in a message attachment, and can contain standard message markup. The content will automatically collapse if it contains 700+ characters or 5+ linebreaks, and will display a "Show more..." link to expand the content. Links posted in the text field will not unfurl.
	 */
	Text string `json:"text,omitempty"`
	/**
	 * Fields are defined as an array, and hashes contained within it will be displayed in a table inside the message attachment.
	 */
	Fields []*Field `json:"fields,omitempty"`
	/**
	 * A valid URL to an image file that will be displayed inside a message attachment. We currently support the following formats: GIF, JPEG, PNG, and BMP.
	 * Large images will be resized to a maximum width of 400px or a maximum height of 500px, while still maintaining the original aspect ratio.
	 */
	ImageURL string `json:"image_url,omitempty"`
	/**
	 * A valid URL to an image file that will be displayed as a thumbnail on the right side of a message attachment. We currently support the following formats: GIF, JPEG, PNG, and BMP.
	 * The thumbnail's longest dimension will be scaled down to 75px while maintaining the aspect ratio of the image. The filesize of the image must also be less than 500 KB.
	 * For best results, please use images that are already 75px by 75px.
	 */
	ThumbURL string `json:"thumb_url,omitempty"`
	/**
	 * Add some brief text to help contextualize and identify an attachment. Limited to 300 characters, and may be truncated further when displayed to users in environments with limited screen real estate.
	 */
	Footer string `json:"footer,omitempty"`
	/**
	 * To render a small icon beside your footer text, provide a publicly accessible URL string in the footer_icon field. You must also provide a footer for the field to be recognized.
	 * We'll render what you provide at 16px by 16px. It's best to use an image that is similarly sized.
	 * Example: "https://platform.slack-edge.com/img/default_application_icon.png"
	 */
	FooterIcon string `json:"footer_icon,omitempty"`
	/**
	 * Does your attachment relate to something happening at a specific time
	 * By providing the ts field with an integer value in "epoch time", the attachment will display an additional timestamp value as part of the attachment's footer.
	 * Use ts when referencing articles or happenings. Your message will have its own timestamp when published.
	 * Example: Providing 123456789 would result in a rendered timestamp of Nov 29th, 1973.
	 */
	TS int64 `json:"ts,omitempty"`
	/**
	 * An array of field names that should be formatted by mrkdwn syntax.
	 */
	MrkdwnIn []string `json:"mrkdwn_in,omitempty"`
}

// WebhookPayload is Slack incoming webhook message
//
// https://api.slack.com/incoming-webhooks
type WebhookPayload struct {
	/**
	 * Text of the message to send.
	 */
	Text string `json:"text,omitempty"`
	/**
	 * Channel, private group, or IM channel to send message to. Can be an encoded ID, or a name.
	 */
	Channel string `json:"channel,omitempty"`
	/**
	 * Set your bot's user name.
	 */
	Username string `json:"username,omitempty"`
	/**
	 * Emoji to use as the icon for this message. Overrides icon_url.
	 */
	IconEmoji string `json:"icon_emoji,omitempty"`
	/**
	 * URL to an image to use as the icon for this message.
	 */
	IconURL string `json:"icon_url,omitempty"`
	/**
	 * Attaching content and links to messages
	 * https://api.slack.com/docs/message-attachments
	 */
	Attachments []*Attachment `json:"attachments,omitempty"`
}
