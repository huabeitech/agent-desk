import PostalMime from 'postal-mime';

export interface Env {
  DESK_WEBHOOK_URL: string;
  DESK_WEBHOOK_SECRET: string;
}

export default {
  async email(message: ForwardableEmailMessage, env: Env, ctx: ExecutionContext): Promise<void> {
    try {
      const rawEmail = await new Response(message.raw).arrayBuffer();
      const parser = new PostalMime();
      const parsed = await parser.parse(rawEmail);

      const fromEmail = message.from || parsed.from?.address || '';
      const fromName = parsed.from?.name || '';
      const toEmail = message.to || (parsed.to && parsed.to[0]?.address) || 'help@crove.com';
      const toName = (parsed.to && parsed.to[0]?.name) || '';
      const subject = message.headers.get('subject') || parsed.subject || '';
      const messageId = message.headers.get('message-id') || parsed.messageId || '';
      const inReplyTo = message.headers.get('in-reply-to') || parsed.inReplyTo || '';
      const references = message.headers.get('references') || (Array.isArray(parsed.references) ? parsed.references.join(' ') : parsed.references) || '';

      const payload = {
        from: fromEmail,
        from_name: fromName,
        to: toEmail,
        to_name: toName,
        subject: subject,
        text: parsed.text || '',
        html: parsed.html || '',
        message_id: messageId,
        in_reply_to: inReplyTo,
        references: references,
      };

      const webhookUrl = env.DESK_WEBHOOK_URL || 'https://desk.crove.com/api/third/email/webhook';
      const webhookSecret = env.DESK_WEBHOOK_SECRET || '';

      const response = await fetch(webhookUrl, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Webhook-Secret': webhookSecret,
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        console.error(`Failed to forward email to webhook: ${response.status} ${response.statusText}`);
      } else {
        console.log(`Successfully forwarded email from ${fromEmail} to Crove Desk`);
      }
    } catch (error) {
      console.error('Error processing inbound email in worker:', error);
    }
  },
};
