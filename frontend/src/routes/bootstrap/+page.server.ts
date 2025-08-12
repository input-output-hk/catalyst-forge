import type { Actions } from './$types';
import { getApiBaseUrl } from '$lib/config.server';

export const actions: Actions = {
  init: async ({ request, fetch }) => {
    try {
      const form = await request.formData();
      const token = String(form.get('token') || '').trim();
      const email = String(form.get('email') || '').trim();
      if (!token) {
        return { error: 'Missing bootstrap token' };
      }
      if (!email) {
        return { error: 'Missing admin email' };
      }

      const API = getApiBaseUrl();

      // 1) Bootstrap → get invite { id, token }
      const r1 = await fetch(`${API}/auth/bootstrap`, {
        method: 'POST',
        headers: { 'content-type': 'application/json', accept: 'application/json' },
        body: JSON.stringify({ email, bootstrap_token: token })
      });
      if (!r1.ok) {
        const body = await r1.text().catch(() => '');
        return { error: `Bootstrap failed: ${r1.status} ${body}` };
      }
      const invite = await r1.json();

      // 2) Device init using invite { token, id }
      const r2 = await fetch(`${API}/auth/devices/init`, {
        method: 'POST',
        headers: { 'content-type': 'application/json', accept: 'application/json' },
        body: JSON.stringify({ token: invite.token, invite_id: invite.id }),
        credentials: 'include'
      });
      if (!r2.ok) {
        const body = await r2.text().catch(() => '');
        return { error: `Device init failed: ${r2.status} ${body}` };
      }
      const init = await r2.json();

      return { invite, init, auto: true };
    } catch (e: any) {
      console.error('bootstrap/init action error', e);
      return { error: `Internal error: ${e?.message ?? e}` };
    }
  }
};


