import { ADMIN, DEWAN } from '../accounts.ts';
import { loginOfficial } from './kit.ts';
let cached = null;
/** Token admin untuk menyiapkan data. Satu login per spec (login dibatasi 10/menit). */
export function adminToken() {
    cached ??= loginOfficial(ADMIN.email, ADMIN.password);
    return cached;
}
export const dewanToken = () => loginOfficial(DEWAN.email, DEWAN.password);
export const operatorToken = () => loginOfficial('operator@silat.id', 'DemoPass_2026');
