## Issues
### Backend Tab
- [ ] regions predefined data with drop down where you can select multiple ones, they are both listed as selected after closing the drop down and are highlighted in the drop down menu
- [ ] The language and framework should not be an option in Monolith services, first time in that tab should first define language and framework and then the possibility to add services, as they all use the same language and framework
- [ ] Make it so you can select multiple auth strategies and token storage options. Make it so you can select multiple options in the drop down menu.
- [ ] make it possible to add technologies to a service like for example websocket for a websocket service
- [ ] The auth subtab should have the possibilty to add roles for auth models that use it
### Data Tab
- [ ] Fix bug where the sub tab switches when typing in the Databases subtab. The h and l keybinds are still active while typing
- [ ] Created Databases should be available as an option in a drop down menu and you can select multiple ones in the domains creating databases field
- [ ] It should be able to add data fields when creating a domain. You should be able to add data fields by writing their names seperated with a coma ,
- [ ] The new relation ship needs an update:<br/>
      So the updated relationship form would simplify to:

Field	Input<br/>
Related Domain	Select from other domains<br/>

Relationship Type	One-to-One · One-to-Many · Many-to-Many<br/>
Cascade Behavior	Cascade delete · Set null · Restrict · No action<br/>
And behind the scenes:

One-to-One / Many-to-One: auto-adds {related_domain}_id as an attribute on the "many" side<br/>
One-to-Many: auto-adds {current_domain}_id on the related domain (it's just the inverse)<br/>
Many-to-Many: auto-creates a junction domain with both foreign keys, editable like any domain<br/>
The Foreign Key Field row gets removed entirely from the user-facing form. If someone needs a non-standard name (say assigned_agent_id instead of user_id), you could expose an optional "customize FK name" toggle that's collapsed by default — keeps the common case clean while still allowing overrides.

- [ ] In the new relationship tab the related domain should be a drop down menu of all the other domains
- [ ] Cascade options should be a drop down menu where you can select the cascade type
- [ ] It should be possible to add multiple Caching strategies
- [ ] All the options should be drop down menus
- [ ] The entities drop down menu should be capable of selecting multiple domains
- [ ] It should be possible to specify the invalidation ttl lenghth in minutes
- [ ] The file storage sub tab should all have drop down selectors
- [ ] The file storage should have the possibility to define filetypes allowed by writing the file extension names with a coma seperated

### Contracts Tab
- [ ] All the options with preset values should have a drop down menu in the contracts tab! Make sure each one has one
- [ ] The source domaims should be a drop down of the existing domains where you can select multiple domains
- [ ] The added domains should automatically have the fields of the domains added, which you can edit delete and still add additional ones
- [ ] The service unit should be a drop down menu of the all the created services
- [ ] The request_dto should be a drop down menu of all the created dtos
- [ ] The response_dto should be a drop down menu of all the created dtos

### Frontend Tab
- [ ] All the predefined values options should be a drop down menu in the frontend tab
- [ ] There should be a a colors option in the theming subtab section where you can paste RGB, HEX or choose multiple which appear in the drop down menu with show casing the color, which adds them as a HEX code
- [ ] There should be a description box in the theming subtab
 - [ ] There should be a vibe option in the theming subtab, where you can set the mood of the theme, so for example giving of a professional vibe playful and so on
 - [ ] In the pages subtab it should be possible to add authorization options for roles which can you select to have access this page. Should be a drop down menu where you can select multiple roles, if a role using authz_model has been selected in the auth sub_tab in the backend tab.
 - [ ] You should be able to declare that a page has paths for other pages, that exist so add a drop down field where you can select multiple other pages it links to


### Infra Tab
- [ ] All the predefined values options should be a drop down menu in the infra tab
- [ ] You should be able to navigate between the options in the subtabs of infra to configure the options with j and k!


### Crosscut Tab
- [ ] All the predefined values options should be a drop down menu in the crosscut tab
- [ ] You should be able to navigate between the options in the subtabs of crosscut to configure the options with j and k!













When creating attributes by adding them in the edit screen they are deleted when exiting and entering the screen again and when writing them in the bar.

The entities should be a drop down of the available domains where you can select multiple of


The multiple choice in auth should be able to select values by pressing enter and space, currently it closes the drop down menu

attr_names is saved but is not displayed only when entering the attributes edit page


Add the possability of specifying the cache length ttl for the invalidation option in the caching subtab

The fields of the source domains in a dto are not added!

Not capable of navigating between options in the sub tabs of Infra and Crosscut tabs






---
When set to backend mircroservices architecture:

 - Add a external data sources configuration field in the services subtab of the backend tab set to microservices
 - In the comm subtab make the from to drop down fields where you can select services already created when creating a communication
 - There should be a possibility to choose a domain in the communication subtab when creating a new link and show show all the domains created as options in a drop down menu


~~In architecture options with the messaging sub tab the domain option in the new event tab should be a drop down menu with all the domains created~~ ✓ done
 
add the possibility of refresh token in the auth subtab of the backend




### CONTROL
- Add the capability to jump down lines with vim like motions for example 3 + jumps down three lines
- Add the capability to use gg to go to the top of the menu and G for gowing to the bottom of the menu







Add inspirations where you can upload images that contain screen shots and mood boards to guide the style of the frontend


Make all the drop down fields in the infra and cross cut subpages with predefines values or values set somewhere else a drop down menu!







