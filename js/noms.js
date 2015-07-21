var store = require('./noms_store.js')
var decode = require('./decode.js')

module.exports = {
  getRoot: store.getRoot,
  getChunk: store.getChunk,
  readValue: decode.readValue,
  getRef: decode.getRef
};

