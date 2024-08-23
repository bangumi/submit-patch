// ==UserScript==
// @name Auto mark all episode as watched when marked subject as watched
// @include       /^https://[^/]*/ep/[^/]*$/
// @include       /^https://[^/]*/subject/[^/]*/edit$/
// ==/UserScript==

(() => {
    const path = document.location.pathname

    if (path.startsWith("/ep/")) {
        const m = /\/ep\/(\d+)/.exec(path);

        $('#columnEpA > h2.title').append(
            `<small><a href="https://patch.bgm38.tv/suggest-episode?episode_id=${m[1]}" class="l" target="_blank">[提供修改建议]</a></small>`
        );
    } else if (path.startsWith('/subject/')) {
        const m = /\/subject\/(\d+)\/edit/.exec(path);

        const el = $('#columnInSubjectA > a:contains("[修改]")')

        el.after(
            `<a href="https://patch.bgm38.tv/suggest?subject_id=${m[1]}" class="l rr" target="_blank">[提供修改建议]</a>`
        );
    }
})();
