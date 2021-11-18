# bot params
| key | required | type | description |
|-|-|-|-|
| bot.token | true | string | Telegram api bot token |
| bot.publicURL | false | string | This url will be used for create registration links |

# OpenID
Bot supports integration with OIDC. It allows administrators control users registration via OpenID provider, used in their org.
| key | required | type | description |
|-|-|-|-|
| oidc.enabled | false | bool | Use OpenID authorization |
| oidc.issuerURL | false | string | OpenID provider issue url |
| oidc.clientID | false | string | OpenID provider client id |
| oidc.clientSecret | false | string | OpenID provider client secret |

# Alertmanager
| key | required | type | description |
|-|-|-|-|
| alertmanager.tag | false | string | Alertmanager docker image tag |
| alertmanager.config | false | string | Initial config of Alertmanager |

# VMAlert
| key | required | type | description |
|-|-|-|-|
| vmalert.tag | false | string | VMAlert docker image tag |
| vmalert.extraArgs | false | string | VMAlert object extraArgs field value |

# VictoriaMetrics
| key | required | type | description |
|-|-|-|-|
| victoriametrics.url | true | string | VictoriaMetrics single instance url, or VictoriaMetrics select instance url for cluster version installation |
