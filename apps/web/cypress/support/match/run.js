/**
 * Menjalankan fungsi async di dalam tes. Di Cypress dibungkus `cy.then` (menunggu promise, dengan timeout
 * sendiri) sehingga urutan perintah tetap benar; di runner Node headless dipanggil langsung.
 * Pakai SATU kali per hook/tes dan letakkan seluruh alur (await berurutan) di dalam fungsinya.
 */
export function runAsync(fn, timeoutMs = 90_000) {
    const c = globalThis.cy;
    return c ? c.then({ timeout: timeoutMs }, () => fn()) : fn();
}
/**
 * Menjalankan beberapa langkah berurutan dalam SATU tes. Langkah boleh mengembalikan promise atau perintah
 * Cypress (mis. `cy.task`); di Cypress tiap langkah diantrekan, di Node ditunggu satu per satu.
 */
export function runSeq(...steps) {
    const c = globalThis.cy;
    if (c) {
        steps.forEach((s) => runAsync(s));
        return undefined;
    }
    return (async () => {
        for (const s of steps)
            await s();
    })();
}
