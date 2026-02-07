import { Client, fetchExchange, subscriptionExchange } from 'urql';
import { createClient as createWSClient } from 'graphql-ws';

const url = '/api/graphql';

// Use the current host for WebSocket connections. In dev, Vite proxies /api
// (including WebSocket) to the backend. In production, the backend serves everything.
const wsUrl = typeof window !== 'undefined'
	? `ws://${window.location.host}/api/graphql`
	: 'ws://localhost/api/graphql';

const wsClient = createWSClient({
	url: wsUrl,
	retryAttempts: Infinity,
	shouldRetry: () => true,
	retryWait: async (retries) => {
		// Exponential backoff: 1s, 2s, 4s, 8s, ... capped at 30s
		const delay = Math.min(1000 * Math.pow(2, retries), 30000);
		await new Promise((resolve) => setTimeout(resolve, delay));
	},
	on: {
		ping: () => console.debug('GraphQL Websocket ping received'),
		connected: () => {
			console.log('Connected to GraphQL Websocket endpoint');
		},
		closed: () => {
			console.log('GraphQL Websocket closed');
		},
		error: (err) => {
			console.error('GraphQL Websocket Error:', err);
		}
	}
});

export const client = new Client({
	url,
	exchanges: [
		fetchExchange,
		subscriptionExchange({
			enableAllOperations: false,
			forwardSubscription(request) {
				const input = { ...request, query: request.query || '' };
				return {
					subscribe(sink) {
						const unsubscribe = wsClient.subscribe(input, sink);
						return { unsubscribe };
					}
				};
			}
		})
	]
});
