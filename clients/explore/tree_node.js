'use strict';

var React = require('react');
var Immutable = require('immutable');
var getRef = require('noms').getRef;

function merge(a, b) {
  var result = {};
  for (var i = 0; i < arguments.length; i++) {
    var obj = arguments[i];
    Object.keys(obj).forEach(function(k) {
      result[k] = obj[k];
    });
  }
  return result;
}

var style = {
  outer: {
    fontFamily: 'Consolas, monospace'
  },
  inner: {
    marginLeft: '20px',
  },
  types: {
    collection: { color: '#bbb', whiteSpace: 'nowrap' },
    string: { color: '#798953' },
    int: { color: '#4562d2' },
    float: { color: '#d28445' },
    boolean: { color: '#75b5aa' }
  },
  contextSpan: {
    color: '#aaa'
  },
  ref: {
    color: '#ddd',
    paddingLeft: '1ex',
    transition: '.3s',
  },
  arrow: {
    display: 'inline-block',
    padding: '0 1ex',
  },
  collapsed: {
    transform: 'rotate(-90deg)',
    WebkitTransform: 'rotate(-90deg)',
  },
  expanded: {},
  treeHeader: {
    whiteSpace: 'nowrap',
  }
};
style.collapsed = merge(style.arrow, style.collapsed);
style.expanded = merge(style.arrow, style.expanded);

var isInteger = Number.isInteger || function(nVal) {
    return typeof nVal === "number" && isFinite(nVal) && nVal > -9007199254740992 && nVal < 9007199254740992 && Math.floor(nVal) === nVal;
}

var TreeNode = React.createClass({
  getInitialState: function() {
    return {
      expand: this.props.expandAll,
      expandAll: this.props.expandAll
    };
  },

  isCollection: function(value) {
    return Immutable.List.isList(value) ||
           Immutable.Set.isSet(value) ||
           Immutable.Map.isMap(value);
  },

  getTypeOf: function(value) {
    if (this.isCollection(value))
      return "collection";
    var type = typeof value;
    if (type == "number") {
      return isInteger(value) ? "int" : "float"
    }
    return type;
  },

  getCollectionName: function(value) {
    if (Immutable.List.isList(value))
      return "List";
    if (Immutable.Set.isSet(value))
      return "Set";
    if (Immutable.Map.isMap(value))
      return "Map";
  },

  valueAsString: function(value) {
    if (this.isCollection(value)) {
      return this.getCollectionName(value) + " (" + value.size + " values)";
    }

    return String(value);
  },

  toggleExpand: function(e) {
    this.setState({
      expand: !this.state.expand,
      expandAll: e.getModifierState("Shift")
    });
  },

  render: function() {
    var value = this.props.value;
    var type = this.getTypeOf(value);
    var isCollection = this.isCollection(value);

    var arrowStyle;
    var arrowContent;
    if (isCollection) {
      arrowStyle = this.state.expand ? style.expanded : style.collapsed;
      arrowContent = '\u25be';
    } else {
      arrowStyle = style.arrow;
      arrowContent = '\u00a0';
    }

    var expander = React.DOM.span({style: arrowStyle}, arrowContent);

    var headerItems = [expander];

    if (this.props.name !== undefined) {
      headerItems.push(React.DOM.span({}, this.props.name + ": "))
    }
    headerItems.push(React.DOM.span({ style: style.types[type] }, this.valueAsString(value)))
    headerItems.push(React.DOM.span({
      className: 'ref',
      style: style.ref
    }, getRef(value)));
    var header = React.DOM.div({
      className: 'tree-header',
      onClick: this.toggleExpand,
      style: style.treeHeader,
    }, headerItems);

    var content = [ header ];
    if (this.state.expand && isCollection) {
      var isSet = Immutable.Set.isSet(value);
      value.forEach(function(subvalue, index) {
        var name = isSet ? undefined : index;
        content.push(TreeNodeFactory({ value: subvalue, name: name, expandAll: this.state.expandAll }));
      }, this);
    }

    return React.DOM.div({ style: style.outer },
      React.DOM.div({ style: style.inner }, content)
    );
  }
});

var TreeNodeFactory = React.createFactory(TreeNode);

module.exports = TreeNode;
