'use strict';

var noms = require('noms');
var React = require('react');
var Map = require('./map.js');

noms.getRoot().then(function(s) {
  return noms.readValue(s, noms.getChunk);
}).then(function(v) {
  return getDatasetRoot(v, 'mlb/heatmap');
}).then(getPitchers).then(renderPitchersList).catch(function(err) {
  console.error(err);
});

function getDatasetRoot(root, id) {
  return root.first().get('value').find(function(map) {
    return map.get('id') === id;
  }).get('root');
}

function getPitchers(datasetRoot) {
  return datasetRoot.first().get('value')
}

var Pitcher = React.createClass({
  render() {
    return <li>
      {this.props.name}
      <Map points={this.props.locations}/>
    </li>;
  }
});

var PitcherList = React.createClass({
  render() {
    var data = this.props.data;
    return <ul>{
      this.props.data.map(function(v, key) {
        return <Pitcher name={key} key={key} locations={v}/>;
      }).toArray()
    }</ul>;
  }
});

function renderPitchersList(list) {
  var target = document.getElementById('heatmap');
  React.render(<PitcherList data={list}/>, target);
}
