mod = angular.module('player')

mod.controller('PlayerCtrl', ($scope, $interval, $http) ->
  'use strict'
  poller = null

  checkPlayerStatus = ->
    console.log('player status poll')
    $http.get('/status').success (data) ->
      console.log(data)
      $scope.playing =
        now: "#{data.Artist} - #{data.Title}"
        # play, pause or stop
        state: data.state
        error: data.error
        progress: Math.floor((parseFloat(data.elapsed)/parseFloat(data.Time))*100)
        track: data.Track
        random: data.random == "1"

  startMonitoring = ->
    poller = $interval(checkPlayerStatus, 1000)

  stopMonitoring = ->
    if angular.isDefined(poller)
      $interval.cancel(poller)
      poller = undefined

  $scope.$on '$destroy', -> $scope.stopMonitoring()

  $scope.play = ->
    $http.get('/play')

  $scope.pause = ->
    $http.get('/pause')

  $scope.previous = ->
    console.log 'previous'
    $http.get('/previous')

  $scope.next = ->
    console.log 'next'
    $http.get('/next')

  $scope.random = (enable) ->
    if enable then $http.get('/randomOn') else $http.get('/randomOff')

  startMonitoring()
)
