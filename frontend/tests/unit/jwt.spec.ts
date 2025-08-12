import { describe, it, expect } from 'vitest';
import { decodeJwtClaims } from '../../src/lib/utils/jwt';

function makeToken(payload: object): string {
	const header = { alg: 'none', typ: 'JWT' };
	const b64 = (obj: object) => Buffer.from(JSON.stringify(obj)).toString('base64url');
	return `${b64(header)}.${b64(payload)}.`;
}

describe('decodeJwtClaims', () => {
	it('returns null for empty token', () => {
		expect(decodeJwtClaims('')).toBeNull();
	});

	it('decodes valid payload', () => {
		const token = makeToken({ sub: '123', email: 'a@b.com' });
		const claims = decodeJwtClaims(token);
		expect(claims).toBeTruthy();
		expect((claims as any).email).toBe('a@b.com');
	});

	it('handles malformed token', () => {
		expect(decodeJwtClaims('abc.def')).toBeNull();
	});
});
