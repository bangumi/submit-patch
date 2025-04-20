// ==UserScript==
// @include       /^https://[^/]*/ep/[^/]*$/
// @include       /^https://[^/]*/subject/[^/]*/edit$/
// @include       /^https://[^/]*/subject/[^/]*$/
// @include       /^https://[^/]*/$/
// ==/UserScript==

(() => {
  const path = document.location.pathname

  if (path === '/') {
    $('h1').append(`&nbsp;<a href="https://patch.bgm38.tv/" target='_blank'><img src="https://patch.bgm38.tv/badge.svg"/></a>`)
    return;
  }

  const episodeMatch = /^\/ep\/(\d+)/.exec(path);
  if (episodeMatch) {
    const episodeID = episodeMatch[1];
    $('#columnEpA > h2.title').append(
      `<small><a href="https://patch.bgm38.tv/edit/episode/${episodeID}" class="l" target="_blank">[提供修改建议]</a></small>`,
      // `<a href="https://patch.bgm38.tv/?type=episode" target='_blank'><img src="https://patch.bgm38.tv/badge/subject/${episodeID}"/></a>`
    );

    return
  }

  const subjectMatch = /^\/subject\/(\d+)/.exec(path);
  if (subjectMatch) {
    const subjectID = subjectMatch[1];
    if (/^\/subject\/\d+\/edit$/.test(path)) {
      $('#columnInSubjectA > a:contains("[修改]")').after(
        `<a href="https://patch.bgm38.tv/edit/subject/${subjectID}" class="l rr" target="_blank">[提供修改建议]</a>`
      );
    } else if (/^\/subject\/\d+$/.test(path)) {
      // $('h1.nameSingle').append(
      //   `<a href="https://patch.bgm38.tv/?type=subject" target='_blank'><img src="https://patch.bgm38.tv/badge/subject/${subjectID}"/></a>`
      // )
    }

    return
  }
})();
