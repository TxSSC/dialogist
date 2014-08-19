(function(root) {
  var SoundBoard = root.SoundBoard = {},
      Models = SoundBoard.Models = {},
      Collections = SoundBoard.Collection = {},
      Views = SoundBoard.Views = {};

  Models.Clip = Backbone.Model.extend({});

  Collections.Clips = Backbone.Collection.extend({
    url: "/clips/",
    model: Models.Clip,

    comparator: function(clip) {
      return clip.get('title').charAt(0).toLowerCase();
    }
  });

  Views.Clip = Backbone.View.extend({
    tagName: 'div',
    className: "clip",

    attributes: function() {
      return {
        "data-view": "clip",
        "data-cid": this.model.cid
      };
    },

    template: _.template("<div><h1><%= title %></h1></div><audio src='<%= location %>' preload='auto'></audio>"),

    initialize: function() {
      this.listenTo(this.model, 'play', this.play);
      this.listenTo(this.model, 'stop', this.stop);
    },

    render: function() {
      this.$el.html(this.template(this.model.toJSON()));

      return this;
    },

    /**
     * Start playing the clip
     */

    play: function() {
      this.$el.addClass("playing");
      this.el.getElementsByTagName("audio")[0].play();
    },

    /**
     * Stop playing the clip
     */

    stop: function() {
      var audio = this.el.getElementsByTagName("audio")[0];

      this.$el.removeClass("playing");
      audio.pause();
      audio.currentTime = 0;
    }
  });

  Views.ClipList = Backbone.View.extend({
    attributes: {
      'data-view': 'clip'
    },

    events: {
      "click [data-view='clip']": "play"
    },

    initialize: function() {
      this.listenTo(this.collection, "add", this.insertClip);
    },

    render: function() {
      var i, len, view, views = [];

      for(i = 0, len = this.collection.length; i < len; i++) {
        view = new Views.Clip({
          model: this.collection.at(i)
        });

        views.push(view.render().el);
      }

      this.$el.append(views);

      return this.el;
    },

    /**
     * Stop playing other children and play the
     * clicked clip
     *
     * @param {jQuery.Event} e
     */

    play: function(e) {
      var next = $(e.currentTarget),
          current = this.$el.find("[data-view='clip'].playing");

      if(current.length) {
        this.collection.get(current.data('cid')).trigger('stop');

        // Just stop if the same clip
        if(next.data('cid') === current.data('cid')) return;
      }

      this.collection.get(next.data('cid')).trigger('play');
    },

    /**
     * Render the clip into the view
     *
     * @param {Backbone.Model} clip
     */

    insertClip: function(clip) {
      var i, len, cid, view;

      view = new Views.Clip({
        model:  clip
      });

      for(i = 0, len = this.collection.length; i < len; i++) {
        cid = this.collection.at(i).cid;

        if(cid === clip.cid) {
          if((i === 0 && len > 1) || i < len - 1) {
            cid = this.collection.at(i + 1).cid;
            cid = this.$el.children("[data-cid='" + cid + "']");
            cid.before(view.render().el);
          } else {
            this.$el.append(view.render().el);
          }

          break;
        }
      }
    }
  });

  /**
   * Initialize new collection
   */

  SoundBoard.Clips = new Collections.Clips();
  SoundBoard.Clips.fetch({
    reset: true,
    success: function() {
      new Views.ClipList({
        el: $(".grid"),
        collection: SoundBoard.Clips
      }).render();
    }
  });

  var events = new EventSource("/events/");

  events.onmessage = function(e) {
    var clip = JSON.parse(e.data);

    SoundBoard.Clips.add(clip);
  };

  return SoundBoard;
})(window)
