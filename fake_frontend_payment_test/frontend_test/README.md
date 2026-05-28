This is a small Next.js frontend used to test the `payment-api-go` Stripe flow.

## What it does

- sends `POST /payment` to the backend
- polls `GET /payment/client-secret/:payment_id`
- shows the payment UUID and the saved client secret
- proxies requests through Next.js to avoid browser CORS issues

## Environment

Create a `.env.local` file in this folder:

```bash
PAYMENT_API_BASE_URL=http://localhost:8080
NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_test_...
```
`PAYMENT_API_BASE_URL` points the proxy routes to your payment API. `NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY` is required by Stripe Elements to create the browser-side payment method.

## Getting Started

First, run the development server:

```bash
npm run dev
# or
yarn dev
# or
pnpm dev
# or
bun dev
```

Open [http://localhost:3000](http://localhost:3000) with your browser to see the result.

You can start editing the page by modifying `app/page.tsx`. The page auto-updates as you edit the file.

### Test flow

1. Start `payment-request-api`, `payment-consumer`, PostgreSQL and RabbitMQ.
2. Run this frontend with `npm run dev`.
3. Fill the form and click `Testar pagamento`.
4. Use the Stripe test card `4242 4242 4242 4242` in the card element.
5. The frontend creates the payment method in Stripe, sends only its ID to the backend, and the consumer confirms the charge.
6. Check the returned `payment_uuid`, `client_secret`, and Stripe dashboard.

This project uses [`next/font`](https://nextjs.org/docs/app/building-your-application/optimizing/fonts) to automatically optimize and load [Geist](https://vercel.com/font), a new font family for Vercel.

## Learn More

To learn more about Next.js, take a look at the following resources:

- [Next.js Documentation](https://nextjs.org/docs) - learn about Next.js features and API.
- [Learn Next.js](https://nextjs.org/learn) - an interactive Next.js tutorial.

You can check out [the Next.js GitHub repository](https://github.com/vercel/next.js) - your feedback and contributions are welcome!

## Deploy on Vercel

The easiest way to deploy your Next.js app is to use the [Vercel Platform](https://vercel.com/new?utm_medium=default-template&filter=next.js&utm_source=create-next-app&utm_campaign=create-next-app-readme) from the creators of Next.js.

Check out our [Next.js deployment documentation](https://nextjs.org/docs/app/building-your-application/deploying) for more details.
