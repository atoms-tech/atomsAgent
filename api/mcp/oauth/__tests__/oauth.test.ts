/**
 * OAuth Token Exchange Test Suite
 *
 * Comprehensive tests for OAuth functionality including
 * encryption, PKCE, state management, and token operations.
 */

import { describe, it, expect, beforeAll, afterAll } from '@jest/globals';
import {
  encrypt,
  decrypt,
  generatePKCE,
  generateState,
  buildAuthorizationUrl,
} from '../helpers';
import crypto from 'crypto';

// Set test encryption key
const TEST_ENCRYPTION_KEY = crypto.randomBytes(32).toString('hex');

beforeAll(() => {
  process.env.OAUTH_ENCRYPTION_KEY = TEST_ENCRYPTION_KEY;
  process.env.GITHUB_CLIENT_ID = 'test_github_client_id';
  process.env.GOOGLE_CLIENT_ID = 'test_google_client_id';
  process.env.AZURE_CLIENT_ID = 'test_azure_client_id';
  process.env.AUTH0_CLIENT_ID = 'test_auth0_client_id';
  process.env.GITHUB_CLIENT_SECRET = 'test_secret';
  process.env.GOOGLE_CLIENT_SECRET = 'test_secret';
  process.env.AZURE_CLIENT_SECRET = 'test_secret';
  process.env.AUTH0_CLIENT_SECRET = 'test_secret';
  process.env.AZURE_AUTH_ENDPOINT = 'https://login.microsoftonline.com/common/oauth2/v2.0/authorize';
  process.env.AUTH0_AUTH_ENDPOINT = 'https://test.auth0.com/authorize';
});

afterAll(() => {
  delete process.env.OAUTH_ENCRYPTION_KEY;
});

describe('Encryption Functions', () => {
  describe('encrypt', () => {
    it('should encrypt plaintext successfully', () => {
      const plaintext = 'test_access_token_12345';
      const encrypted = encrypt(plaintext);

      expect(encrypted).toBeDefined();
      expect(typeof encrypted).toBe('string');
      expect(encrypted).not.toBe(plaintext);
    });

    it('should produce encrypted output with correct format', () => {
      const plaintext = 'test_token';
      const encrypted = encrypt(plaintext);
      const parts = encrypted.split(':');

      expect(parts).toHaveLength(3); // iv:authTag:encrypted
      expect(parts[0]).toHaveLength(32); // 16 bytes IV in hex
      expect(parts[1]).toHaveLength(32); // 16 bytes auth tag in hex
      expect(parts[2].length).toBeGreaterThan(0); // encrypted data
    });

    it('should produce different ciphertext for same plaintext', () => {
      const plaintext = 'test_token';
      const encrypted1 = encrypt(plaintext);
      const encrypted2 = encrypt(plaintext);

      expect(encrypted1).not.toBe(encrypted2);
    });

    it('should throw error with invalid encryption key', () => {
      const originalKey = process.env.OAUTH_ENCRYPTION_KEY;
      process.env.OAUTH_ENCRYPTION_KEY = 'invalid_key';

      expect(() => encrypt('test')).toThrow('Invalid encryption key');

      process.env.OAUTH_ENCRYPTION_KEY = originalKey;
    });
  });

  describe('decrypt', () => {
    it('should decrypt ciphertext successfully', () => {
      const plaintext = 'test_access_token_12345';
      const encrypted = encrypt(plaintext);
      const decrypted = decrypt(encrypted);

      expect(decrypted).toBe(plaintext);
    });

    it('should handle various plaintext lengths', () => {
      const testCases = [
        'short',
        'medium_length_token_value',
        'very_long_token_value_with_lots_of_characters_'.repeat(10),
      ];

      testCases.forEach((plaintext) => {
        const encrypted = encrypt(plaintext);
        const decrypted = decrypt(encrypted);
        expect(decrypted).toBe(plaintext);
      });
    });

    it('should handle special characters', () => {
      const specialChars = 'token!@#$%^&*()_+-=[]{}|;:,.<>?/~`';
      const encrypted = encrypt(specialChars);
      const decrypted = decrypt(encrypted);

      expect(decrypted).toBe(specialChars);
    });

    it('should throw error for invalid encrypted data format', () => {
      expect(() => decrypt('invalid_format')).toThrow('Invalid encrypted data format');
      expect(() => decrypt('part1:part2')).toThrow('Invalid encrypted data format');
    });

    it('should throw error for tampered ciphertext', () => {
      const plaintext = 'test_token';
      const encrypted = encrypt(plaintext);
      const tampered = encrypted.replace(/.$/, '0'); // Change last character

      expect(() => decrypt(tampered)).toThrow();
    });

    it('should throw error with wrong decryption key', () => {
      const plaintext = 'test_token';
      const encrypted = encrypt(plaintext);

      // Change encryption key
      const originalKey = process.env.OAUTH_ENCRYPTION_KEY;
      process.env.OAUTH_ENCRYPTION_KEY = crypto.randomBytes(32).toString('hex');

      expect(() => decrypt(encrypted)).toThrow();

      process.env.OAUTH_ENCRYPTION_KEY = originalKey;
    });
  });
});

describe('PKCE Functions', () => {
  describe('generatePKCE', () => {
    it('should generate code verifier and challenge', () => {
      const { codeVerifier, codeChallenge } = generatePKCE();

      expect(codeVerifier).toBeDefined();
      expect(codeChallenge).toBeDefined();
      expect(typeof codeVerifier).toBe('string');
      expect(typeof codeChallenge).toBe('string');
    });

    it('should generate code verifier with correct length', () => {
      const { codeVerifier } = generatePKCE();

      // Code verifier should be 43-128 characters (32 bytes in base64url = 43 chars)
      expect(codeVerifier.length).toBeGreaterThanOrEqual(43);
      expect(codeVerifier.length).toBeLessThanOrEqual(128);
    });

    it('should generate valid base64url encoded strings', () => {
      const { codeVerifier, codeChallenge } = generatePKCE();

      // base64url should not contain +, /, or =
      expect(codeVerifier).not.toMatch(/[+/=]/);
      expect(codeChallenge).not.toMatch(/[+/=]/);
    });

    it('should generate different values each time', () => {
      const pkce1 = generatePKCE();
      const pkce2 = generatePKCE();

      expect(pkce1.codeVerifier).not.toBe(pkce2.codeVerifier);
      expect(pkce1.codeChallenge).not.toBe(pkce2.codeChallenge);
    });

    it('should generate challenge as SHA256 of verifier', () => {
      const { codeVerifier, codeChallenge } = generatePKCE();

      // Manually compute challenge
      const hash = crypto.createHash('sha256').update(codeVerifier).digest();
      const expectedChallenge = hash.toString('base64url');

      expect(codeChallenge).toBe(expectedChallenge);
    });
  });

  describe('generateState', () => {
    it('should generate random state string', () => {
      const state = generateState();

      expect(state).toBeDefined();
      expect(typeof state).toBe('string');
      expect(state.length).toBeGreaterThan(0);
    });

    it('should generate different values each time', () => {
      const state1 = generateState();
      const state2 = generateState();

      expect(state1).not.toBe(state2);
    });

    it('should generate base64url encoded string', () => {
      const state = generateState();

      // base64url should not contain +, /, or =
      expect(state).not.toMatch(/[+/=]/);
    });

    it('should have sufficient entropy (at least 32 bytes)', () => {
      const state = generateState();

      // 32 bytes in base64url = 43 characters
      expect(state.length).toBeGreaterThanOrEqual(43);
    });
  });
});

describe('Authorization URL Builder', () => {
  describe('buildAuthorizationUrl', () => {
    it('should build GitHub authorization URL', () => {
      const state = 'test_state_123';
      const codeChallenge = 'test_challenge_123';
      const redirectUri = 'https://example.com/callback';
      const scopes = ['repo', 'user'];

      const url = buildAuthorizationUrl('github', state, codeChallenge, redirectUri, scopes);

      expect(url).toContain('https://github.com/login/oauth/authorize');
      expect(url).toContain(`client_id=${process.env.GITHUB_CLIENT_ID}`);
      expect(url).toContain(`state=${state}`);
      expect(url).toContain(`code_challenge=${codeChallenge}`);
      expect(url).toContain('code_challenge_method=S256');
      expect(url).toContain('response_type=code');
      expect(url).toContain(`redirect_uri=${encodeURIComponent(redirectUri)}`);
      expect(url).toContain('scope=repo+user');
    });

    it('should build Google authorization URL', () => {
      const state = 'test_state_123';
      const codeChallenge = 'test_challenge_123';
      const redirectUri = 'https://example.com/callback';
      const scopes = ['openid', 'email', 'profile'];

      const url = buildAuthorizationUrl('google', state, codeChallenge, redirectUri, scopes);

      expect(url).toContain('https://accounts.google.com/o/oauth2/v2/auth');
      expect(url).toContain(`client_id=${process.env.GOOGLE_CLIENT_ID}`);
      expect(url).toContain(`state=${state}`);
      expect(url).toContain('scope=openid+email+profile');
    });

    it('should build Azure authorization URL', () => {
      const state = 'test_state_123';
      const codeChallenge = 'test_challenge_123';
      const redirectUri = 'https://example.com/callback';

      const url = buildAuthorizationUrl('azure', state, codeChallenge, redirectUri);

      expect(url).toContain('https://login.microsoftonline.com');
      expect(url).toContain('oauth2/v2.0/authorize');
      expect(url).toContain(`client_id=${process.env.AZURE_CLIENT_ID}`);
    });

    it('should build Auth0 authorization URL', () => {
      const state = 'test_state_123';
      const codeChallenge = 'test_challenge_123';
      const redirectUri = 'https://example.com/callback';

      const url = buildAuthorizationUrl('auth0', state, codeChallenge, redirectUri);

      expect(url).toContain('https://test.auth0.com/authorize');
      expect(url).toContain(`client_id=${process.env.AUTH0_CLIENT_ID}`);
    });

    it('should throw error for unsupported provider', () => {
      const state = 'test_state';
      const codeChallenge = 'test_challenge';
      const redirectUri = 'https://example.com/callback';

      expect(() => {
        buildAuthorizationUrl('invalid_provider', state, codeChallenge, redirectUri);
      }).toThrow('Unsupported provider');
    });

    it('should properly encode redirect URI', () => {
      const state = 'test_state';
      const codeChallenge = 'test_challenge';
      const redirectUri = 'https://example.com/callback?param=value&other=test';

      const url = buildAuthorizationUrl('github', state, codeChallenge, redirectUri);

      expect(url).toContain(encodeURIComponent(redirectUri));
    });

    it('should include all required parameters', () => {
      const state = 'test_state';
      const codeChallenge = 'test_challenge';
      const redirectUri = 'https://example.com/callback';

      const url = buildAuthorizationUrl('github', state, codeChallenge, redirectUri);

      const requiredParams = [
        'client_id',
        'redirect_uri',
        'state',
        'response_type',
        'code_challenge',
        'code_challenge_method',
      ];

      requiredParams.forEach((param) => {
        expect(url).toContain(param);
      });
    });
  });
});

describe('Token Operations', () => {
  describe('End-to-End Encryption', () => {
    it('should encrypt and decrypt OAuth tokens', () => {
      const tokens = {
        access_token: 'gho_test_access_token_1234567890',
        refresh_token: 'gho_test_refresh_token_0987654321',
      };

      const encryptedAccess = encrypt(tokens.access_token);
      const encryptedRefresh = encrypt(tokens.refresh_token);

      expect(encryptedAccess).not.toBe(tokens.access_token);
      expect(encryptedRefresh).not.toBe(tokens.refresh_token);

      const decryptedAccess = decrypt(encryptedAccess);
      const decryptedRefresh = decrypt(encryptedRefresh);

      expect(decryptedAccess).toBe(tokens.access_token);
      expect(decryptedRefresh).toBe(tokens.refresh_token);
    });

    it('should handle long token values', () => {
      const longToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.' +
        'eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.' +
        'SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c';

      const encrypted = encrypt(longToken);
      const decrypted = decrypt(encrypted);

      expect(decrypted).toBe(longToken);
    });
  });

  describe('PKCE Flow', () => {
    it('should complete PKCE flow correctly', () => {
      // Simulate PKCE flow
      const { codeVerifier, codeChallenge } = generatePKCE();

      // Store code_verifier (encrypted)
      const encryptedVerifier = encrypt(codeVerifier);

      // Later, retrieve and decrypt verifier
      const retrievedVerifier = decrypt(encryptedVerifier);

      expect(retrievedVerifier).toBe(codeVerifier);

      // Verify challenge matches
      const hash = crypto.createHash('sha256').update(retrievedVerifier).digest();
      const expectedChallenge = hash.toString('base64url');

      expect(codeChallenge).toBe(expectedChallenge);
    });
  });
});

describe('Security Tests', () => {
  it('should not expose plaintext in encrypted output', () => {
    const secrets = [
      'very_secret_access_token',
      'super_secret_refresh_token',
      'api_key_12345',
    ];

    secrets.forEach((secret) => {
      const encrypted = encrypt(secret);
      expect(encrypted).not.toContain(secret);
    });
  });

  it('should produce cryptographically secure random values', () => {
    const states = Array.from({ length: 100 }, () => generateState());

    // Check uniqueness
    const uniqueStates = new Set(states);
    expect(uniqueStates.size).toBe(100);

    // Check length consistency
    states.forEach((state) => {
      expect(state.length).toBeGreaterThanOrEqual(43);
    });
  });

  it('should reject tampered encrypted data', () => {
    const plaintext = 'test_token';
    const encrypted = encrypt(plaintext);

    // Tamper with each component
    const parts = encrypted.split(':');

    // Tamper with IV
    const tamperedIV = parts[0].slice(0, -1) + '0' + ':' + parts[1] + ':' + parts[2];
    expect(() => decrypt(tamperedIV)).toThrow();

    // Tamper with auth tag
    const tamperedTag = parts[0] + ':' + parts[1].slice(0, -1) + '0' + ':' + parts[2];
    expect(() => decrypt(tamperedTag)).toThrow();

    // Tamper with ciphertext
    const tamperedCipher = parts[0] + ':' + parts[1] + ':' + parts[2].slice(0, -1) + '0';
    expect(() => decrypt(tamperedCipher)).toThrow();
  });
});

describe('Error Handling', () => {
  it('should handle missing encryption key gracefully', () => {
    const originalKey = process.env.OAUTH_ENCRYPTION_KEY;
    delete process.env.OAUTH_ENCRYPTION_KEY;

    expect(() => encrypt('test')).toThrow('Invalid encryption key');
    expect(() => decrypt('test')).toThrow('Invalid encryption key');

    process.env.OAUTH_ENCRYPTION_KEY = originalKey;
  });

  it('should handle invalid encryption key length', () => {
    const originalKey = process.env.OAUTH_ENCRYPTION_KEY;
    process.env.OAUTH_ENCRYPTION_KEY = 'too_short';

    expect(() => encrypt('test')).toThrow('Invalid encryption key');

    process.env.OAUTH_ENCRYPTION_KEY = originalKey;
  });

  it('should handle malformed encrypted data', () => {
    const malformedData = [
      '',
      'invalid',
      'part1:part2',
      'not:hex:data',
    ];

    malformedData.forEach((data) => {
      expect(() => decrypt(data)).toThrow();
    });
  });
});
