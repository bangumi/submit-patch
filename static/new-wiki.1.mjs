import {
  parseToMap, stringifyMap
} from '@bgm38/wiki';

import platforms from './common/subject_platforms.json' assert {type: 'json'};
import templates from './common/wiki_template.json' assert {type: 'json'};

const subjectType = globalThis.subject_type
const configForSubjectType = platforms.platforms[subjectType];

if (!configForSubjectType) {
  const tpl = templates[platforms.default[subjectType].wiki_tpl];
  $('#infobox').val(tpl);
}

function platformChange() {
  const currentPlatform = $('#wiki-form').find('input[name="platform"]:checked').val();

  const config = platforms.platforms[subjectType]?.[currentPlatform]
  const tpl = templates[config?.wiki_tpl ?? platforms.default[subjectType].wiki_tpl];

  const currentInfobox = $('#infobox').val()

  const defaultWiki = parseToMap(tpl)
  const w = parseToMap(currentInfobox)

  defaultWiki.type == defaultWiki.type || w.type;

  w.data.forEach((v, k) => {
    defaultWiki.data.set(k, v);
  })

  $('#infobox').val(stringifyMap(defaultWiki));
}

$('.platform-radio').on('change', platformChange)
