const DIGITS = '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz';

function indexOf(ch: string): number {
	const i = DIGITS.indexOf(ch);
	return i >= 0 ? i : 0;
}

function toIndices(s: string): number[] {
	return Array.from(s).map((ch) => indexOf(ch));
}

function fromIndices(indices: number[]): string {
	return indices.map((i) => DIGITS[Math.max(0, Math.min(61, i))]).join('');
}

function padRight(s: string, length: number): string {
	while (s.length < length) s += DIGITS[0];
	return s;
}

function midpoint(a: string, b: string): string {
	const maxLen = Math.max(a.length, b.length);
	const padA = padRight(a, maxLen);
	const padB = padRight(b, maxLen);
	const digitsA = toIndices(padA);
	const digitsB = toIndices(padB);

	for (let i = 0; i < maxLen; i++) {
		if (digitsA[i] < digitsB[i] - 1) {
			const mid = Math.floor((digitsA[i] + digitsB[i]) / 2);
			const result = digitsA.slice(0, i);
			result.push(mid);
			return fromIndices(result);
		}
		if (digitsA[i] === digitsB[i]) continue;

		// Adjacent digits — go deeper
		const result = digitsA.slice(0, i + 1);
		const nextA = i + 1 < digitsA.length ? digitsA[i + 1] : 0;
		result.push(Math.floor((nextA + 62) / 2));
		return fromIndices(result);
	}

	// Equal up to maxLen — extend
	const result = digitsA.slice();
	result.push(31);
	return fromIndices(result);
}

function incrementKey(key: string): string {
	const digits = toIndices(key);
	for (let i = digits.length - 1; i >= 0; i--) {
		if (digits[i] < 61) {
			const result = digits.slice(0, i);
			let mid = Math.floor((digits[i] + 62) / 2);
			if (mid === digits[i]) mid = digits[i] + 1;
			result.push(mid);
			return fromIndices(result);
		}
	}
	digits.push(31);
	return fromIndices(digits);
}

function decrementKey(key: string): string {
	const digits = toIndices(key);
	for (let i = digits.length - 1; i >= 0; i--) {
		if (digits[i] > 1) {
			const result = digits.slice(0, i);
			result.push(Math.floor(digits[i] / 2));
			return fromIndices(result);
		}
		if (digits[i] === 1) {
			const result = digits.slice(0, i);
			result.push(0, 31);
			return fromIndices(result);
		}
	}
	if (key.length > 1) return key.slice(0, -1);
	return '0';
}

/**
 * Generate an order key between two existing keys.
 * Pass empty string for either to generate at the start/end.
 */
export function orderBetween(a: string, b: string): string {
	if (!a && !b) return 'V';
	if (!a) return decrementKey(b);
	if (!b) return incrementKey(a);
	return midpoint(a, b);
}
