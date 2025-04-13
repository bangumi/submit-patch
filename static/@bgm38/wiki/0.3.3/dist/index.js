// src/error.ts
var GlobalPrefixError = "missing prefix '{{Infobox' at the start";
var GlobalSuffixError = "missing suffix '}}' at the end";
var ArrayNoCloseError = "array should be closed by '}'";
var ArrayItemWrappedError = "array item should be wrapped by '[]'";
var ExpectingNewFieldError = "missing '|' to start a new field";
var ExpectingSignEqualError = "missing '=' to separate field name and value";
var WikiSyntaxError = class extends Error {
  lino;
  line;
  constructor(lino, line, message) {
    super(toErrorString(lino, line, message));
    this.line = line;
    this.lino = lino;
  }
};
function toErrorString(lino, line, msg) {
  if (line === null) {
    return `WikiSyntaxError: ${msg}, line ${lino}`;
  }
  return `WikiSyntaxError: ${msg}, line ${lino}: ${JSON.stringify(line)}`;
}

// src/shared.ts
var prefix = "{{Infobox";
var suffix = "}}";

// src/types.ts
var WikiArrayItem = class {
  k;
  v;
  constructor(k, v) {
    this.k = k || void 0;
    this.v = v;
  }
};
var WikiItem = class {
  key;
  value;
  array;
  values;
  constructor(key, value, type) {
    this.key = key;
    switch (type) {
      case "array": {
        this.array = true;
        this.values = [];
        break;
      }
      case "object": {
        this.value = value;
        break;
      }
    }
  }
  convertToArray() {
    if (this.array) {
      return;
    }
    this.array = true;
    this.values ??= [];
    this.values.push(new WikiArrayItem(void 0, this.value));
    this.value = void 0;
  }
  push(item) {
    var _a, _b;
    this.convertToArray();
    if (item.array) {
      (_a = this.values) == null ? void 0 : _a.push(...item.values ?? []);
    } else {
      (_b = this.values) == null ? void 0 : _b.push(new WikiArrayItem(void 0, item.value));
    }
  }
};

// src/stringify.ts
var stringifyArray = (arr) => {
  if (!arr) {
    return "";
  }
  return arr.reduce((pre, item) => `${pre}
[${item.k ? `${item.k}|` : ""}${item.v ?? ""}]`, "");
};
function stringify(wiki) {
  const body = wiki.data.reduce((pre, item) => {
    if (item.array) {
      return `${pre}
|${item.key} = {${stringifyArray(item.values)}
}`;
    }
    return `${pre}
|${item.key} = ${item.value ?? ""}`;
  }, "");
  const type = wiki.type ? " " + wiki.type : "";
  return `${prefix}${type}${body}
${suffix}`;
}
function stringifyMap(wiki) {
  const body = [...wiki.data].map(([key, value]) => {
    if (typeof value === "string") {
      return `|${key} = ${value ?? ""}`;
    }
    return `|${key} = {${stringifyArray(value)}
}`;
  }).join("\n");
  const type = wiki.type ? " " + wiki.type : "";
  return `${prefix}${type}
${body}
${suffix}`;
}

// src/index.ts
function parseToMap2(s) {
  try {
    return [null, parseToMap(s)];
  } catch (error) {
    if (error instanceof WikiSyntaxError) {
      return [error, null];
    }
    throw error;
  }
}
function parseToMap(s) {
  const w = parse(s);
  const data = /* @__PURE__ */ new Map();
  for (const item of w.data) {
    let previous = data.get(item.key);
    if (previous) {
      if (typeof previous === "string") {
        previous = [new WikiArrayItem(void 0, previous)];
      }
      if (item.array) {
        previous.push(...item.values);
      } else {
        previous.push(new WikiArrayItem(void 0, item.value));
      }
      data.set(item.key, previous);
      continue;
    }
    if (item.array) {
      data.set(item.key, item.values ?? []);
    } else {
      data.set(item.key, item.value);
    }
  }
  return { type: w.type, data };
}
function processInput(s) {
  let offset = 2;
  s = s.replaceAll("\r\n", "\n");
  for (const char of s) {
    switch (char) {
      case "\n": {
        offset++;
        break;
      }
      case " ":
      case "	": {
        continue;
      }
      default: {
        return [s.trim(), offset];
      }
    }
  }
  return [s.trim(), offset];
}
function parse2(s) {
  try {
    return [null, parse(s)];
  } catch (error) {
    if (error instanceof WikiSyntaxError) {
      return [error, null];
    }
    throw error;
  }
}
function parse(s) {
  var _a, _b, _c, _d;
  const wiki = {
    type: "",
    data: []
  };
  const [strTrim, offset] = processInput(s);
  if (strTrim === "") {
    return wiki;
  }
  if (!strTrim.startsWith(prefix)) {
    throw new WikiSyntaxError(offset - 1, null, GlobalPrefixError);
  }
  if (!strTrim.endsWith(suffix)) {
    throw new WikiSyntaxError((((_a = s.match(/\n/g)) == null ? void 0 : _a.length) ?? -2) + 1, null, GlobalSuffixError);
  }
  const arr = strTrim.split("\n");
  if (arr[0]) {
    wiki.type = parseType(arr[0]);
  }
  const fields = arr.slice(1, -1);
  let inArray = false;
  for (let i = 0; i < fields.length; ++i) {
    const line = (_b = fields[i]) == null ? void 0 : _b.trim();
    const lino = offset + i;
    if (!line) {
      continue;
    }
    if (line.startsWith("|")) {
      if (inArray) {
        throw new WikiSyntaxError(lino, line, ArrayNoCloseError);
      }
      const meta = parseNewField(lino, line);
      inArray = meta[2] === "array";
      const field = new WikiItem(...meta);
      wiki.data.push(field);
    } else if (inArray) {
      if (line.startsWith("}")) {
        inArray = false;
        continue;
      }
      if (i === fields.length - 1) {
        throw new WikiSyntaxError(lino, line, ArrayNoCloseError);
      }
      (_d = (_c = wiki.data.at(-1)) == null ? void 0 : _c.values) == null ? void 0 : _d.push(new WikiArrayItem(...parseArrayItem(lino, line)));
    } else {
      throw new WikiSyntaxError(lino, line, ExpectingNewFieldError);
    }
  }
  return wiki;
}
var parseType = (line) => {
  if (!line.includes("}}")) {
    return line.slice(prefix.length).trim();
  }
  return line.slice(prefix.length, line.indexOf("}}")).trim();
};
var parseNewField = (lino, line) => {
  const str = line.slice(1);
  const index = str.indexOf("=");
  if (index === -1) {
    throw new WikiSyntaxError(lino, line, ExpectingSignEqualError);
  }
  const key = str.slice(0, index).trim();
  const value = str.slice(index + 1).trim();
  switch (value) {
    case "{": {
      return [key, "", "array"];
    }
    default: {
      return [key, value, "object"];
    }
  }
};
var parseArrayItem = (lino, line) => {
  if (!line.startsWith("[") || !line.endsWith("]")) {
    throw new WikiSyntaxError(lino, line, ArrayItemWrappedError);
  }
  const content = line.slice(1, -1);
  const index = content.indexOf("|");
  if (index === -1) {
    return ["", content.trim()];
  }
  return [content.slice(0, index).trim(), content.slice(index + 1).trim()];
};
export {
  ArrayItemWrappedError,
  ArrayNoCloseError,
  ExpectingNewFieldError,
  ExpectingSignEqualError,
  GlobalPrefixError,
  GlobalSuffixError,
  WikiArrayItem,
  WikiItem,
  WikiSyntaxError,
  parse,
  parse2,
  parseToMap,
  parseToMap2,
  stringify,
  stringifyMap
};
