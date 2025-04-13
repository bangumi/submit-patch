import { parseToMap, stringifyMap } from '../@bgm38/wiki/0.3.3/dist/index.js';
import platforms from "../common/subject_platforms.9b43ea.json" assert {type: 'json'};
import templates from "/static/common/wiki_template.eacd4d.json" assert {type: 'json'};

const subjectType = globalThis.subject_type;
const configForSubjectType = platforms.platforms[subjectType];

if (!configForSubjectType) {
  const tpl = templates[platforms.default[subjectType].wiki_tpl];
  $('#infobox').val(tpl);
}

function platformChange() {
  const currentPlatform = $('#wiki-form').find('input[name="platform"]:checked').val();

  const config = platforms.platforms[subjectType]?.[currentPlatform]
  const tpl = templates[config?.wiki_tpl ?? platforms.default[subjectType].wiki_tpl];

  const infobox = $('#infobox')

  const currentInfobox = infobox.val()

  const defaultWiki = parseToMap(tpl)
  const w = parseToMap(currentInfobox)
  const finalWiki = parseToMap(tpl)

  finalWiki.type = w.type || defaultWiki.type;

  w.data.forEach((v, k) => {
    if (v !== defaultWiki.data.get(k)) {
      finalWiki.data.set(k, v);
    }
  })

  infobox.val(stringifyMap(finalWiki));
}

$('.platform-radio').on('change', platformChange)
