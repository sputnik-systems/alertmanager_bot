# Pre-install
If you want use this, you should:
* Install and use one of operators: [VictoriaMetrics](https://github.com/VictoriaMetrics/operator) or [Prometheus](https://github.com/prometheus-operator/prometheus-operator) oprator. Right now bot works only with VMRules or PrometheusRules object types.
* Register telegram bot account.

# Installation
You can install it over [helm-chart](../deployments/helm-chart) templates.

# Post-install
After installation complete, you can send commands to bot.

Subscribtion is possible only for "registered" users. It's mean that this chat id must be exists in receivers list. Before registration will be completed, bot will be response to any command with same message:

<img src="images/register.png" alt="register" width="500"/>

Following by given link will be add your chat id in receivers list.

## Subscribtion
Subscribtions example:

<img src="images/subscribe1.png" alt="subscribe" width="500"/>

When you pressed any keyboard key, bot will be remove keyboard:

<img src="images/subscribe2.png" alt="subscribe" width="500"/>

## Disable subscribtion
Disable subscribtions example:

<img src="images/unsubscribe1.png" alt="unsubscribe" width="500"/>

After button pressing:

<img src="images/unsubscribe2.png" alt="unsubscribe" width="500"/>
