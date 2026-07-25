---
title: StarData Cloud vs StarData Developer 
sidebar_label: StarData Cloud vs StarData Developer 
sidebar_position: 10
hide_table_of_contents: false
---

## What is StarData Cloud and StarData Developer?

StarData offers two unique but complementary experiences within our broader product suite, **StarData Cloud** and **StarData Developer**.

As the name suggests, StarData _Developer_ is designed with the developer in mind, where project development actually occurs. StarData Developer is meant for the primary developers of project assets and dashboards, allowing them to import, wrangle, iterate on, and explore the data before presenting it for broader consumption by the team. StarData Developer is meant to run on your local machine - see here for some [recommendations and best practices](/developers/tutorials/performance#local-development--stardata-developer) - but it is a simple process to [deploy a project](/developers/deploy/deploy-dashboard) once ready to StarData Cloud.


StarData Cloud, on the other hand, is designed for our dashboard consumers and allows broader team members to easily collaborate. Once the developer has deployed the dashboard onto StarData Cloud, these users will be able to utilize the dashboards to interact with their data, set alerts / bookmarks, investigate nuances / anomalies, or otherwise perform everyday tasks for their business needs at StarData speed.

## Is StarData Cloud a higher offering than StarData Developer?

Based on the naming, it might be confusing and easy to assume that StarData Cloud is our "higher" offering but **that is not the case!** Similarly, StarData Developer is _not meant to be used as a standalone tool either_.

StarData Developer and StarData Cloud are to be used together. StarData Developer provides a space for our developers to define and test any new or needed changes to the data and/or dashboards before pushing to our StarData Cloud users, who need stable access to working dashboards. Then, once finalized, these dashboards are deployed to the StarData Cloud project for broader consumption and to power business use cases.
:::info Isn't StarData Developer enough?

Please note that a common **misnomer** is that StarData Developer can be a sufficient replacement for StarData Cloud. They both serve different purposes but are meant to be used _in conjunction_. StarData enables speed of exploration and is easy to use for developers, allowing the project to be iterated on quickly. StarData Cloud then allows for shared collaboration at scale, especially for production deployments.

:::


### Why deploy to StarData Cloud?

StarData Developer is an extremely strong tool for deep-diving into your data, as it allows users to import sources from many destinations and join these tables together to create something useful in a slice-and-dice visualization. Many times the feedback we receive is, "`I didn't even know my data had an issue in it,`" or, "`In a few minutes, I was able to make new insights into my data that would've taken me hours.`" This is great, but if that's possible in StarData Developer, why publish the dashboard to StarData Cloud? 


<div style={{ 
  position: "relative", 
  width: "100%", 
  paddingTop: "56.25%", 
  borderRadius: "15px",  /* Softer corners */
  boxShadow: "0px 4px 15px rgba(0, 0, 0, 0.2)"  /* Shadow effect */
}}>
  <iframe credentialless="true"
    src="https://www.youtube.com/embed/zW1Xms2qQlc?si=OpKVKN7csHCY_AcX"
    frameBorder="0"
    allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
    allowFullScreen
    style={{
      position: "absolute",
      top: 0,
      left: 0,
      width: "100%",
      height: "100%",
      borderRadius: "10px"
    }}
  ></iframe>
</div>
<br />



## StarData Developer

StarData Developer is designed around developers. Using a familiar IDE-like interface, developers are able to import data, create SQL models, and create metrics views. Many of the underlying files in StarData Developer are either SQL or YAML files. Once data is imported into StarData (and the underlying OLAP engine), developers are able to perform last-mile ETL changes using one or a series of SQL models (as their own [DAG](https://en.wikipedia.org/wiki/Directed_acyclic_graph#:~:text=A%20directed%20acyclic%20graph%20is,a%20path%20with%20zero%20edges)). You can then create and materialize your ["One Big Table"](/developers/build/models/models-101#one-big-table-and-dashboarding) for your dashboard needs. Finally, any specifications for your dimensions and measures can be defined and tested in Developer's dashboard preview.

![Empty Project](/img/concepts/rcvsrd/empty-project.png)


## StarData Cloud

Once the dashboard has been [deployed to StarData Cloud](/developers/deploy/deploy-dashboard), the dashboard can be shared with others and viewed by other members of your StarData Cloud organization. As you can see below, the UI is different from Developer. Upon accessing StarData Cloud, a user will be able to view all the projects they have been granted access to by project admins. 


![StarData Cloud Projects](/img/concepts/rcvsrd/rill-cloud-projects.png)

 After selecting a specific project, they will be directed to a list of dashboards. From StarData Cloud, the dashboard consumer does not have the ability to make any modifications to sources or models. However, they are given some additional capabilities that are not accessible in StarData Developer, such as alerting, creating bookmarks or shareable public URLs, checking the project status, and more.

 :::info Dashboard 101

 For more details about using a StarData Cloud dashboard, please refer to our [Explore section](/guide/dashboards/explore)!

 :::
